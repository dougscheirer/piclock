package main

import "fmt"
import "time"
import "runtime"
import "flag"
// import "io"
import "io/ioutil"
import "github.com/buger/jsonparser"
// import "strings"

// piclock -config={config file}

type Alarm struct {
	name string
	time time.Time
}

type Settings struct {
	countdownTime time.Duration
	sleepTime time.Duration
	alarmPath string
}

func logMessage(msg string) {
	// TODO: log to a file?
	_, fname, line, _ := runtime.Caller(1)
	fmt.Printf("%s: %s(%d): %s\n", time.Now().Format(time.UnixDate), fname, line, msg)
}

func defaultSettings() Settings {
	var s Settings
	
	s.countdownTime, _ = time.ParseDuration("1m")
	s.sleepTime, _ = time.ParseDuration("10ms")
	s.alarmPath = "/etc/default/piclock/alarms"

	return s
}

func getString(data []byte, name string) string {
	s, err := jsonparser.GetString(data, name)
	if err == nil {
		logMessage(fmt.Sprintf("%s : %s", name, s))
		return s
	}
	return ""
}

func getDuration(data []byte, name string) time.Duration {
	duration, err := jsonparser.GetString(data, name)
	if err == nil {
		d, err := time.ParseDuration(duration)
		if err == nil { 
			logMessage(fmt.Sprintf("%s : %s", name, duration))
			return d 
		} else {
			logMessage(fmt.Sprintf("bad value '%s' : %s", duration, err.Error()))
			return -1
		}
	} else {
		return -1
	}
}

func settingsFromJSON(s Settings, data []byte) Settings {
	countdown := getDuration(data, "countdownTime")
	sleepTime := getDuration(data, "sleepTime")
	alarmPath := getString(data, "alarmPath")

	if countdown >= 0 { s.countdownTime = countdown }
	if sleepTime >= 0 { s.sleepTime 		= sleepTime }
	if alarmPath != "" { s.alarmPath   = alarmPath }
	return s
}

func initSettings() Settings {
	logMessage("initSettings")

	// defaults
	s := defaultSettings()

	// define our flags first
	configFile := flag.String("config", "/etc/default/piclock/piclock.conf", "config file path")

	// parse the flags
	flag.Parse()

	// try to open it
	data, err := ioutil.ReadFile(*configFile)
	if err != nil {
		logMessage(fmt.Sprintf("Could not load conf file '%s', using defaults", *configFile))
		return s
	}

	logMessage(fmt.Sprintf("Reading configuration from '%s'", *configFile))

	// json parse it
	return settingsFromJSON(s, data)
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
	logMessage("readAlarmCache")
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

func main() {
	/*
		Main app
		    startup: initialization lcd/alarms
	*/
	settings := initSettings()
	initLCD()
	initAlarms()

	// loop:
	loop := true
	for loop {
		time.Sleep(settings.sleepTime)
		// Read cache dir every 1(?) secs in table
		alarmTable := readAlarmCache();
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

		  // get the next alarm that is going to fire
		  nextAlarm := getNextAlarm(alarmTable);
		  duration := nextAlarm.time.Sub(time.Now());

		  if (duration > 0) {
			  // start a countdown?
		  	if (duration < settings.countdownTime) {
		  		setCountdownMode()
		  	}
		  } else {
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
