package main

import "fmt"
import "time"
import "runtime"
import "flag"
import "io"
import "io/ioutil"
import "encoding/json"
import "strings"

// piclock -config={config file}

type Alarm struct {
	name string
	time time.Time
}

type Settings struct {
	countdownTime time.Duration
}

func logMessage(msg string) {
	// TODO: log to a file?
	_, fname, line, _ := runtime.Caller(1)
	fmt.Printf("%s: %s(%d): %s\n", time.Now().Format(time.UnixDate), fname, line, msg)
}

func defaultSettings() Settings {
	var s Settings
	s.countdownTime = 60
	return s
}

func settingsFromJSON(s Settings, data []byte) Settings {
	type Message struct {
		key, val string
	}
	logMessage(string(data))
	dec := json.NewDecoder(strings.NewReader(string(data)))
	for {
		var m Message
		if err := dec.Decode(&m); err == io.EOF {
			break
		} else if err != nil {
			logMessage(err.Error())
			return s
		}
		logMessage(fmt.Sprintf("%s: %s", m.key, m.val))
		switch {
			case m.key == "countdown" :
				d, err := time.ParseDuration(m.val)
				if err != nil { s.countdownTime = d }
			default:
				logMessage(fmt.Sprintf("unknown key '%s'", m.key))
				break
		}
	}
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
		time.Sleep(500 * time.Millisecond)
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
