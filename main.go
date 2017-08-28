package main

import "fmt"
import "time"

type Alarm struct {
	name string
	time time.Time
}

func pizza() { }
func initAlarms() bool { return true }

func initLCD() bool { return true }
func readAlarmCache() []Alarm { 
	var ret = [1]Alarm{}
 	return ret
}

func mainButtonPressed() bool { return false }
func clearAlarmCacheFiles() { }
func runCalendarRefresh() { }
func isAlarming() bool { return false }
func endAlarm() { }
func setClockMode() { }
func disableFirstAlarm(table []Alarm) { }
func nextAlarm(table []Alarm) { }
func getNextAlarm(table []Alarm) { }

func main() {
	/*
		Main app
		    startup: initialization lcd/alarms
	*/
	initLCD();
	initAlarms();

	// loop:
	var loop bool = true
	for loop {
		// Read cache dir every 1(?) secs in table
		var alarmTable []Alarm = readAlarmCache();
		// If button press
		if mainButtonPressed() {
		  // If table.empty
		  if len(alarmTable) == 0 {
		   	// Clear cache directory
		   	clearAlarmCacheFiles();
		   	// run calendar job
		   	runCalendarRefresh();
 			  // Else if alarm.active
		  } else if isAlarming() {
		    // Alarm.disable
		    endAlarm();
		    // Set clock mode
		    setClockMode();
			} else {	
		  	// Table.findfirstenabled.disable (to file)
		  	disableFirstAlarm(alarmTable);
		  }
		    
		  time.Time now = time.Time.now();
		  // get the next alarm that is going to fire
		  nextAlarm = getNextAlarm(alarmTable);
		  // start a countdown?
		  if (nextAlarm.time - now() < countdownTime) {
		  	setCountdownMode();
		  }
		  
		  // start the alarm?
		  if (nextAlarm.time <= now()) {
		    // Set alarm mode
		    setAlarmMode();
		    disableAlarm(nextAlarm);
		  }
		  
		  // update UI
		  updateAlarmLEDs();
		  updateExtraLEDs();
		} 
	}
}