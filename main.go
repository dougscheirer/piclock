package main

import (
	"fmt"
	"time"
	"piclock/sevenseg_backpack"
	"sync"
	"errors"
	"strings"
	"encoding/json"
	"io/ioutil"
	"os"
	// gpio lib
	"github.com/stianeikeland/go-rpio"
)

var wg sync.WaitGroup

// piclock -config={config file}

// Note: fields must be capitalized or json.Marshal will not convert them
type Alarm struct {
	Id 			string
	Name 		string
	When 		time.Time
	Effect  string
	disabled bool 	// set to true when we're checking alarms and it fired
	countdown bool  // set to true when we're checking alarms and we signaled countdown
}

type Effect struct {
	id string  // TODO: a struct to tell the effects generator what to do
	val interface{}
}

type ButtonInfo struct {
	pressed 	bool
	duration 	time.Duration
}

func initAlarms(settings *Settings) bool {
	logMessage("initAlarms")
	return true
}

func readAlarmCache() []Alarm {
	// logMessage("readAlarmCache")
	ret := make([]Alarm, 0, 100)
 	return ret
}

func mainButtonPressed() Effect {
	return Effect{ id:"mainButton", val : ButtonInfo{pressed: true}  }
}

func mainButtonReleased(d time.Duration) Effect {
	return Effect{ id:"mainButton", val : ButtonInfo{pressed: false, duration: d} }
}

func setCountdownMode(alarm Alarm) Effect {
	return Effect{id:"countdown", val: alarm}
}

func setAlarmMode(alarm Alarm) Effect {
	return Effect{id:"alarm", val: alarm}
}

func updateAlarmLEDs() {}
func updateExtraLEDs() {}

func alarmError() Effect {
	return Effect{ id: "alarmError" }
}

func writeAlarms(alarms []Alarm, fname string) error {
	output, err := json.Marshal(alarms)
	logMessage(string(output))
	if err != nil {
		return err
	}
	return ioutil.WriteFile(fname, output, 0644)
}

func handledAlarm(alarm Alarm, handled map[string]Alarm) bool {
	// if the time changed, consider it "unhandled"
	v, ok := handled[alarm.Id]
	if !ok { return false}
	if v.When != alarm.When { return false }
	// everything else ignore
	return true
}

func cacheFilename(settings *Settings) string {
	return settings.GetString("alarmPath") + "/alarm.json"
}

func getAlarmsFromService(settings *Settings, handled map[string]Alarm) ([]Alarm, error) {
	alarms := make([]Alarm, 0)
	srv := GetCalenderService(settings.GetString("secretPath"))

	// TODO: if it wasn't available, send an Alarm message
	if srv == nil {
		return alarms, errors.New("Failed to get calendar service")
	}

	// map the calendar to an ID
	calName := settings.GetString("calendar")
	var id string
	{
		logMessage("get calendar list")
		list, err := srv.CalendarList.List().Do()
		if err != nil {
			logMessage(err.Error())
			return alarms, err
		}
		for _, i := range list.Items {
			if i.Summary == calName {
				id = i.Id
				break
			}
		}
	}

	if id == "" {
		return alarms, errors.New(fmt.Sprintf("Could not find calendar %s", calName))
	}
	// get next 10 (?) alarms
	t := time.Now().Format(time.RFC3339)
	events, err := srv.Events.List(id).
										ShowDeleted(false).
										SingleEvents(true).
										TimeMin(t).
										MaxResults(10).
										OrderBy("startTime").
										Do()
	if err != nil {
		return alarms, err
	}

	// remove the cached alarms if they are present
	cacheFile := cacheFilename(settings)
	if _, err := os.Stat(cacheFile); !os.IsNotExist(err) {
		err = os.Remove(cacheFile)
		// an error here is a system config issue
		if err != nil {
			// TODO: severe error effect
			logMessage(fmt.Sprintf("Error: %s", err.Error()))
			return alarms, err
		}
	}

	// calculate the alarms, write to a file
	if len(events.Items) > 0 {
	  for _, i := range events.Items {
	    // If the DateTime is an empty string the Event is an all-day Event.
	    // So only Date is available.
	    if i.Start.DateTime == "" {
	  		logMessage(fmt.Sprintf("Not a time based alarm: %s @ %s", i.Summary, i.Start.Date))
	    	continue
	    }
	    var when time.Time
	    when, err = time.Parse(time.RFC3339, i.Start.DateTime)
	    if err != nil {
	    	// skip bad formats
	    	logMessage(err.Error())
	    	continue
	    }

	    if when.Sub(time.Now()) < 0 {
	    	logMessage(fmt.Sprintf("Skipping old alarm: %s", i.Id))
	    	continue
	    }

	    alm := Alarm{Id: i.Id, Name: i.Summary, When: when, disabled: false}

	    // look for hastags (does not work ATM, the gAPI is broken I think)
	    music := strings.Contains(i.Summary, "#music")
	    random := strings.Contains(i.Summary, "#random")
	    file := strings.Contains(i.Summary, "#file")
	    tones := strings.Contains(i.Summary, "#tone")	// tone or tones

	    // priority is arbitrary except for random (default)
	    if music {
	    	alm.Effect = "music"
	    } else if file {
	     	alm.Effect = "file" // TODO: figure out the filename
	    }	else if tones {
	    	alm.Effect = "tones" // TODO: tone options
	    } else if random {
	    	alm.Effect = "random"
			}	else {
	    	alm.Effect = "random"
	    }

	    // has this one been handled?
	    if handledAlarm(alm, handled) {
	    	logMessage(fmt.Sprintf("Skipping handled alarm: %s", alm.Id))
	    }

	    alarms = append(alarms, alm)
	  }

	  // cache in a file for later if we go offline
	  writeAlarms(alarms, cacheFile)
	}

	return alarms, nil
}

func getAlarmsFromCache(settings *Settings, handled map[string]Alarm) ([]Alarm, error) {
	alarms := make([]Alarm, 0)
	data, err := ioutil.ReadFile(cacheFilename(settings))
	if err != nil {
		return alarms, err
	}
	err = json.Unmarshal(data, &alarms)
	if err != nil {
		return alarms, err
	}
	// remove any that are in the "handled" map
	for i:=len(alarms)-1;i>=0;i-- {
		if handledAlarm(alarms[i], handled) {
			// remove is append two slices without the part we don't want
			logMessage(fmt.Sprintf("Discard handled alarm: %s", alarms[i].Id))
			alarms = append(alarms[:i], alarms[i+1:]...)
		}
	}

	return alarms, nil
}

func getAlarms(settings *Settings, cA chan Alarm, cE chan Effect, cH chan Alarm) {
	defer wg.Done()

	// keep a list of things that we have done
	// TODO: GC the list occasionally
	handledAlarms := map[string]Alarm{}

	for true {
		// read any handled alarms first
		keepReading := true;
		for keepReading {
			select {
				case alm := <- cH:
					handledAlarms[alm.Id] = alm
				default:
					keepReading = false
					logMessage("No handled alarms")
			}
		}

		alarms, err := getAlarmsFromService(settings, handledAlarms)
		if err != nil {
			cE <- alarmError()
			logMessage(err.Error())
			// try the backup
			alarms, err = getAlarmsFromCache(settings, handledAlarms)
			if err != nil {
				// very bad, so...delete and try again later?
				// TODO: more effects
				fmt.Printf("Error reading alarm cache: %s\n", err.Error())
				time.Sleep(time.Second)
			}
		}

		// tell cA that we have some alarms
		cA <- Alarm{}	// reset hack
		for i:=0;i<len(alarms);i++ {
			cA <- alarms[i]
		}

		time.Sleep(settings.GetDuration("alarmRefreshTime"))
	}
}

func replaceAtIndex(in string, r rune, i int) string {
  out := []rune(in)
  out[i] = r
  return string(out)
}

func toButtonInfo(val interface{}) (*ButtonInfo, error) {
	switch v:=val.(type) {
	case ButtonInfo:
		return &v, nil
	default:
		return nil, errors.New(fmt.Sprintf("Bad type: %T", v))
	}
}

func toBool(val interface{}) (bool, error) {
	switch v := val.(type) {
		case bool:
			return v, nil
		default:
			return false, errors.New(fmt.Sprintf("Bad type: %T", v))
	}
}

func toInt(val interface{}) (int, error) {
	switch v := val.(type) {
		case int:
			return v, nil
		default:
			return -1, errors.New(fmt.Sprintf("Bad type: %T", v))
	}
}

func toAlarm(val interface{}) (*Alarm, error) {
	switch v := val.(type) {
	case Alarm:
		return &v, nil
	default:
		return nil, errors.New(fmt.Sprintf("Bad type: %T", v))
	}
}

func toString(val interface{}) (string, error) {
	switch v := val.(type) {
	case string:
		return v, nil
	default:
		return "", errors.New(fmt.Sprintf("Bad type: %T", v))
	}
}

func displayClock(display *sevenseg_backpack.Sevenseg) {
	// standard time display
	colon := "15:04"
	now := time.Now()
	if now.Second() % 2 == 0 {
		// no space required for the colon
		colon = "1504"
	}

	timeString := now.Format(colon)
	if timeString[0] == '0' {
		timeString = replaceAtIndex(timeString, ' ', 0)
	}

	err := display.Print(timeString)
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
	}
}

func displayCountdown(display *sevenseg_backpack.Sevenseg, alarm *Alarm) bool {
	// calculate 10ths of secs to alarm time
	count := alarm.When.Sub(time.Now()) / (time.Second/10)
	if count > 9999 {
		count = 9999
	} else if count <= 0 {
		return false
	}
	s := fmt.Sprintf("%d.%d", count / 10, count % 10)
	var blinkRate uint8 = sevenseg_backpack.BLINK_OFF
	if count < 100 {
		blinkRate = sevenseg_backpack.BLINK_2HZ
	}
	display.SetBlinkRate(blinkRate)
	display.Print(s)
	return true
}

func runEffects(settings *Settings, c chan Effect) {
	defer wg.Done()

	display, err := sevenseg_backpack.Open(
		settings.GetByte("i2c_device"),
		settings.GetInt("i2c_bus"),
		settings.GetBool("i2c_simulated"))

	if err != nil {
		fmt.Printf("Error: %s", err.Error())
		return
	}

	// turn on LED dump?
	display.DebugDump(settings.GetBool("debug_dump"))

	display.SetBrightness(3)
	// ready to rock
	display.DisplayOn(true)

	var mode string = "clock"
	var countdown *Alarm
	var error_id = 0
	alarmSegment := 0
	DEFAULT_SLEEP := settings.GetDuration("sleepTime")
	sleepTime := DEFAULT_SLEEP

	for true {
		var e Effect
		select {
		case e = <-c:
			switch e.id {
				case "debug":
					v, _ := toBool(e.val)
					display.DebugDump(v)
				case "clock":
					mode = e.id
				case "countdown":
					mode = e.id
					countdown, _ = toAlarm(e.val)
					sleepTime = 10 * time.Millisecond
				case "alarmError":
					// TODO: alarm error LED
					mode = e.id
					display.Print("Err")
					error_id, _ = toInt(e.val)
				case "terminate":
					fmt.Printf("terminate")
					return
				case "print":
					v, _ := toString(e.val)
					display.Print(v)
				case "alarm":
					mode = e.id
					alm, _ := toAlarm(e.val)
					sleepTime = 10*time.Millisecond
					fmt.Printf(">>>>>>>>>>>>>>> ALARM <<<<<<<<<<<<<<<<<<\n%s %s %s\n", alm.Name, alm.When, alm.Effect)
				case "mainButton":
					info, _ := toButtonInfo(e.val)
					if info.pressed {
						logMessage("Main button pressed")
						if (mode == "alarm") {
							// TODO: cancel the alarm
							mode = "clock"
							sleepTime = DEFAULT_SLEEP
						}
					} else {
						logMessage(fmt.Sprintf("Main button released: %ds", info.duration))
					}
				default:
					fmt.Printf("Unhandled %s\n", e.id)
			}
		default:
			// nothing?
			time.Sleep(time.Duration(sleepTime))
		}

		switch mode {
			case "clock":
				displayClock(display)
			case "countdown":
				if !displayCountdown(display, countdown) {
					mode = "clock"
					sleepTime = DEFAULT_SLEEP
				}
			case "alarmError":
				fmt.Sprintf("Error: %d\n", error_id)
				display.Print("Err")
			case "output":
				// do nothing
			case "alarm":
				// do a strobing 0, light up segments 0 - 5
				display.RefreshOn(false)
				display.SetBlinkRate(sevenseg_backpack.BLINK_OFF)
				display.ClearDisplay()
				for i:=0;i<4;i++ {
					display.SegmentOn(byte(i), byte(alarmSegment), true)
				}
				display.RefreshOn(true)
				alarmSegment = (alarmSegment + 1) % 6
			default:
				logMessage(fmt.Sprintf("Unknown mode: '%s'\n", mode))
		}
	}

	display.DisplayOn(false)
}

func checkAlarm(settings *Settings, cA chan Alarm, cE chan Effect, cH chan Alarm) {
	defer wg.Done()

	alarms := make([]Alarm, 0)
	var lastLogSecond = -1

	for true {
		// try reading from our channel
		keepReading := true
		alarmsRead := 0
		for keepReading {
			select {
				case alm := <- cA :
					alarmsRead++
					if alm.Name == "" {
						// reset the list
						logMessage("Reset alarm list")
						alarms = make([]Alarm, 0)
					} else {
						logMessage(fmt.Sprintf("Alarm: %+v", alm))
						alarms = append(alarms, alm)
					}
				default:
					keepReading = false
			}
		}

		// alarms come in sorted with soonest first
	  for index:=0;index<len(alarms);index++ {
	  	if alarms[index].disabled {
	  		continue // skip processed alarms
	  	}

	  	now := time.Now()
	  	duration := alarms[index].When.Sub(now)
	  	if lastLogSecond != now.Second() && now.Second() % 30 == 0 {
	  		lastLogSecond = now.Second()
	  		logMessage(fmt.Sprintf("Time to next alarm: %ds (%ds to countdown)", duration / time.Second, (duration - settings.GetDuration("countdownTime"))/time.Second))
	  	}

		  if (duration > 0) {
			  // start a countdown?
			  countdown := settings.GetDuration("countdownTime")
		  	if (duration < countdown) {
		  		cE <- setCountdownMode(alarms[0])
		  		alarms[index].countdown = true
		  	}
		  } else {
		    // Set alarm mode
				cE <- setAlarmMode(alarms[index])
				// let someone know we handled it
				cH <- alarms[index]
				alarms[index].disabled = true
		  }
		  break
		}
		// take some time off
		time.Sleep(100 * time.Millisecond)
	}
}

func toggleDebugDump(on bool) Effect {
	return Effect{ id: "debug", val: on }
}

func confirm_calendar_auth(settings *Settings, c chan Effect) {
	defer func(){ c <- toggleDebugDump(settings.GetBool("debug_dump")) }()

	c <- toggleDebugDump(false)
	c <- Effect{id: "print", val: "...."}
	for true {
		c := GetCalenderService(settings.GetString("secretPath"))
		if c != nil { return }
		// TODO: set some error indicators
	}
}

func watchButtons(settings *Settings, cE chan Effect) {
	defer wg.Done()

	simulated := settings.GetString("button_simulated")

	for true {
		if len(simulated) != 0 {
				// TODO: map buttons to keys

			} else {
				// map ports to buttons
				err := rpio.Open()
				if err != nil {
					logMessage(err.Error())
					return
				}

				// TODO: configurable pin numbers and high or low
				// picking GPIO 4 results in collisions with I2C operations
				pin := rpio.Pin(25)

				// for now we only care about the "low" state
				pin.Input()        // Input mode
				pin.PullUp()			 // GND => button press
				pressed := false
				pressTime := time.Now()

				for true {
					res := pin.Read()  // Read state from pin (High / Low)
					if !pressed {
						if res == 0 {
							pressTime = time.Now()
						 	cE <- mainButtonPressed()
							pressed = true
						}
					} else {
						if res == 1 {
							cE <- mainButtonReleased(time.Now().Sub(pressTime))
							pressed = false
						}
					}
					time.Sleep(10*time.Millisecond)
				}
			}
	}
}

func main() {
	// read config information
	settings := InitSettings()

	// dump them (debugging)
	fmt.Println("\n>>> Settings <<<\n")
	settings.Dump()
	fmt.Println("\n>>> Settings <<<\n")

	/*
		Main app
		    startup: initialization HW/alarms
	*/
	initAlarms(settings)

  alarmChannel := make(chan Alarm, 1)
  effectChannel := make(chan Effect, 1)
  handledChannel := make(chan Alarm, 1)

	// wait on our three workers: alarm fetcher, clock runner, alarm checker, button checker
  wg.Add(4)

  // start the effect thread so we can update the LEDs
	go runEffects(settings, effectChannel)

	// google calendar requires OAuth access, so make sure we get it
	// before we go into the main loop
	confirm_calendar_auth(settings, effectChannel)

	go getAlarms(settings, alarmChannel, effectChannel, handledChannel)
	go checkAlarm(settings, alarmChannel, effectChannel, handledChannel)
	go watchButtons(settings, effectChannel)

	wg.Wait()
}
