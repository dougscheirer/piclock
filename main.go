package main

import (
	"fmt"
	"time"
	"piclock/sevenseg_backpack"
	"sync"
	"errors"
)

var wg sync.WaitGroup

// piclock -config={config file}

type Alarm struct {
	name string
	time time.Time
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
func setCountdownMode() { }
func setAlarmMode() { }
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

func getAlarms(settings *Settings, cA chan Alarm, cE chan Effect) {
	defer wg.Done()

	for true {
		srv := GetCalenderService()
		// TODO: if it wasn't available, send an Alarm message
		if srv == nil {
			cE <- alarmError()
			// TODO: use the cached alarms
			time.Sleep(time.Second)
			continue			
		}

		// get next 10 alarms
		t := time.Now().Format(time.RFC3339)
		events, err := srv.Events.List("primary").
											ShowDeleted(false).
											SingleEvents(true).
											TimeMin(t).
											MaxResults(10).
											OrderBy("startTime").
											Do()
		if err != nil {
		 	cE <- alarmError()
			// TODO: use the cached alarms
			time.Sleep(time.Second)
			continue			
		}

		fmt.Printf("Upcoming events: %T\n", events.Items)
		if len(events.Items) > 0 {
		  for _, i := range events.Items {
		  	if !i.Summary.Contains("#piclock")  {
		  		continue
		  	}
		    var when string
		    // If the DateTime is an empty string the Event is an all-day Event.
		    // So only Date is available.
		    if i.Start.DateTime != "" {
		      when = i.Start.DateTime
		    } else {
		      when = i.Start.Date
		    }
		    fmt.Printf("%s (%s)\n", i.Summary, when)
		  }
		} else {
		  fmt.Printf("No upcoming events found.\n")
		}

		// TODO: signal that the alarms got refreshed?
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

	display.Print("8888")
	display.SetBrightness(3)
	// ready to rock
	display.DisplayOn(true)

	var mode string = "clock"
	var countdown = 0

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
					fmt.Printf("alarmError")
				case "terminate":
					fmt.Printf("terminate")
					return
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
			default:
				fmt.Printf("Unknown mode: '%s'", mode)
		}
	}
	
	display.DisplayOn(false)
}

func checkAlarm(settings *Settings, c0 chan Alarm, c1 chan Effect) {
	defer wg.Done()

	// loop:
	loop := true
	for loop {
		time.Sleep(settings.GetDuration("sleepTime"))
		// Read cache dir every 1(?) secs in table
		alarmTable := readAlarmCache()
		// If button press
		if mainButtonPressed() {
		  // If table.empty
		  if len(alarmTable) == 0 {
		   	// Clear cache directory
		   	clearAlarmCacheFiles()
		   	// run calendar job
		   	runCalendarRefresh()
 			  // Else if alarm.active
		  } else if isAlarming() {
		    // Alarm.disable
		    endAlarm()
		    // Set clock mode
		    setClockMode()
			} else {
		  	// Table.findfirstenabled.disable (to file)
		  	disableFirstAlarm(alarmTable)
		  }

		  // get the next alarm that is going to fire
		  nextAlarm := getNextAlarm(alarmTable)
		  duration := nextAlarm.time.Sub(time.Now())

		  if (duration > 0) {
			  // start a countdown?
		  	if (duration < settings.GetDuration("countdownTime")) {
		  		setCountdownMode()
		  	}
		  } else {
		    // Set alarm mode
		    setAlarmMode()
		    disableAlarm(nextAlarm)
		  }

		  // update UI
		  updateAlarmLEDs()
		  updateExtraLEDs()
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
