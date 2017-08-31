package main

import (
	"fmt"
	"time"
	// "io"
	// "strings"
	"piclock/sevenseg_backpack"
)

// piclock -config={config file}

type Alarm struct {
	name string
	time time.Time
}

func initAlarms() bool {
	logMessage("initAlarms")
	return true
}

func initLCD() bool {
	logMessage("initLCD")
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

func getAlarms(settings *Settings) {
	for true {
		reconcileAlarms(settings.GetString("alarmPath"))
		// TODO: signal that the alarms got refreshed?
		time.Sleep(settings.GetDuration("alarmRefreshTime"))
	}
}

func replaceAtIndex(in string, r rune, i int) string {
	    out := []rune(in)
	    out[i] = r
	    return string(out)
}

func runClock(settings *Settings) {
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

func main() {
	/*
		Main app
		    startup: initialization lcd/alarms
	*/
	settings := InitSettings()
	// dump them (debugging)
	fmt.Println("Settings:")
	settings.Dump()

	initLCD()
	initAlarms()

	go getAlarms(settings)
	go runClock(settings)

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
