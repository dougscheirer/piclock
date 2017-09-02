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
)

var wg sync.WaitGroup

// piclock -config={config file}

// Note: fields must be capitalized or json.Marshal will not convert them
type Alarm struct {
	Name 		string
	When 		time.Time
	Effect  string
	disabled bool
}

type Effect struct {
	id string  // TODO: a struct to tell the effects generator what to do
	val interface{}
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

func mainButtonPressed() bool { return false }
func clearAlarmCacheFiles() { }
func runCalendarRefresh() { }
func isAlarming() bool { return false }
func endAlarm() { }
func setClockMode() { }

func setCountdownMode(duration time.Duration) Effect {
	return Effect{id:"countdown", val: duration}
}
func setAlarmMode(alarm Alarm) Effect {
	return Effect{id:"alarm", val: alarm}
}

func disableAlarm(a Alarm) { }
func updateAlarmLEDs() {}
func updateExtraLEDs() {}
func disableFirstAlarm(table []Alarm) { }
func nextAlarm(table []Alarm) { }
func getNextAlarm(table []Alarm) Alarm { return table[0] }

func reconcileAlarms(path string) {
	// TODO: get alarms from calendar, remove ones that don't exist
}

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

func cacheFilename(settings *Settings) string {
	return settings.GetString("alarmPath") + "/alarm.json"
}

func getAlarmsFromService(settings *Settings) ([]Alarm, error) {
	alarms := make([]Alarm, 0)
	srv := GetCalenderService()
	// TODO: if it wasn't available, send an Alarm message
	if srv == nil {
		return alarms, errors.New("Failed to get calendar service")
	}

	// map the calendar to an ID
	calName := settings.GetString("calendar")
	var id string
	{
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
	    	logMessage(fmt.Sprintf("Already passed the time for: %s (%s)\n", i.Summary, i.Start.DateTime))
	    	continue
	    }

	    alm := Alarm{Name: i.Summary, disabled: false, When: when}

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

	    alarms = append(alarms, alm)
	  }

	  // cache in a file for later if we go offline
	  writeAlarms(alarms, cacheFile)
	}

	return alarms, nil
}

func getAlarmsFromCache(settings *Settings) ([]Alarm, error) {
	alarms := make([]Alarm, 0)
	data, err := ioutil.ReadFile(cacheFilename(settings))
	if err != nil {
		return alarms, err
	}
	err = json.Unmarshal(data, &alarms)
	return alarms, err
}

func getAlarms(settings *Settings, cA chan Alarm, cE chan Effect) {
	defer wg.Done()

	for true {
		alarms, err := getAlarmsFromService(settings)
		if err != nil {
			cE <- alarmError()
			logMessage(err.Error())
			// try the backup
			alarms, err = getAlarmsFromCache(settings)
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

func displayCountdown(display *sevenseg_backpack.Sevenseg, count int) {
	display.Print(fmt.Sprintf("%d", count))
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
	display.DebugDump(settings.GetBool("i2c_simulated"))

	display.SetBrightness(3)
	// ready to rock
	display.DisplayOn(true)

	var mode string = "clock"
	var countdown = 0
	var error_id = 0

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
					countdown, _ = toInt(e.val)
				case "alarmError":
					// TODO: alarm error LED
					mode = e.id
					display.Print("Err")
					error_id, _ = toInt(e.val)
				case "terminate":
					fmt.Printf("terminate")
					return
				case "alarm":
					alm, _ := toAlarm(e.val)
					fmt.Printf(">>>>>>>>>>>>>>> ALARM <<<<<<<<<<<<<<<<<<\n%s %s %s\n", alm.Name, alm.When, alm.Effect)
				default:
					fmt.Printf("Unhandled %s\n", e.id)
			}
		default:
			// nothing?
			time.Sleep(250 * time.Millisecond)
		}

		switch mode {
			case "clock":
				displayClock(display)
			case "countdown":
				displayCountdown(display, countdown)
				countdown--
				if countdown < 0 { mode = "clock" }
			case "alarmError":
				fmt.Sprintf("Error: %d\n", error_id)
			default:
				fmt.Printf("Unknown mode: '%s'", mode)
		}
	}

	display.DisplayOn(false)
}

func checkAlarm(settings *Settings, cA chan Alarm, cE chan Effect) {
	defer wg.Done()

	alarms := make([]Alarm, 0)

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
						alarms = make([]Alarm, 0)
					} else {
						alarms = append(alarms, alm)
					}
				default:
					keepReading = false
			}
		}

		if alarmsRead > 0 {
			logMessage(fmt.Sprintf("%+v\n", alarms))
		}

		// alarms come in sorted with soonest first
	  for index:=0;index<len(alarms);index++ {
	  	if alarms[index].disabled {
	  		continue
	  	}

	  	duration := alarms[index].When.Sub(time.Now())

		  if (duration > 0) {
			  // start a countdown?
			  countdown := settings.GetDuration("countdownTime")
		  	if (duration < countdown) {
		  		cE <- setCountdownMode(duration)
		  	}
		  } else {
		    // Set alarm mode
				cE <- setAlarmMode(alarms[index])
		    alarms[index].disabled = true
		  }
		  break
		}
	}
}

func toggleDebugDump(on bool) Effect {
	return Effect{ id: "debug", val: !on }
}

func confirm_calendar_auth(settings *Settings, c chan Effect) {
	defer func(){ c <- toggleDebugDump(false) }()

	c <- toggleDebugDump(true)
	for true {
		c := GetCalenderService()
		if c != nil { return }
		// TODO: set some error indicators
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

	// wait on our three workers: alarm fetcher, clock runner, alarm checker
  wg.Add(3)

	go runEffects(settings, effectChannel)

	// google calendar requires OAuth access, so make sure we get it
	// before we go into the main loop
	confirm_calendar_auth(settings, effectChannel)

	go getAlarms(settings, alarmChannel, effectChannel)
	go checkAlarm(settings, alarmChannel, effectChannel)

	wg.Wait()
}
