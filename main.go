package main

import (
	"fmt"
	"time"
	"piclock/sevenseg_backpack"
	"sync"
)

var wg sync.WaitGroup

// piclock -config={config file}

type Alarm struct {
	name string
	time time.Time
}

type Effect struct {
	data string  // TODO: a struct to tell the effects generator what to do
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

func getAlarms(settings *Settings, c chan Alarm) {
	defer wg.Done()

	for true {
		srv := GetCalenderService()
		// TODO: if it wasn't available, send an Alarm message
		fmt.Printf("srv: %T\n", srv)
		// get next 10 alarms
		t := time.Now().Format(time.RFC3339)
		events, err := srv.Events.List("primary").ShowDeleted(false).
		SingleEvents(true).TimeMin(t).MaxResults(10).OrderBy("startTime").Do()
		if err != nil {
		  // log.Fatalf("Unable to retrieve next ten of the user's events. %v", err)
		}

		fmt.Println("Upcoming events:")
		if len(events.Items) > 0 {
		  for _, i := range events.Items {
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

	for true {
		time.Sleep(250 * time.Millisecond)
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
		// fmt.Printf("%d : %s %s\n", now.Second(), colon, timeString)
		err := display.Print(timeString)
		if err != nil {
			fmt.Printf("Error: %s\n", err.Error())
		}
	}
	// never get here, the above loop is "forever"
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

	// wait on our three workers: alarm fetcher, clock runner, alarm checker
  wg.Add(3)

  alarmChannel := make(chan Alarm, 1)
  effectChannel := make(chan Effect, 1)

	go getAlarms(settings, alarmChannel)
	go runEffects(settings, effectChannel)
	go checkAlarm(settings, alarmChannel, effectChannel)

	wg.Wait()
}
