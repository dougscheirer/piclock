package main

import (
	"log"
	"os"
	"sync"
	"time"
)

// piclock -config={config file}

var wg sync.WaitGroup

var features = []string{}

func main() {
	// read config information
	settings := initSettings()

	// are we just generating the oauth token?
	if settings.GetBool("oauth") {
		confirmCalendarAuth(settings)
		return
	}

	// first try to set up the log (optional)
	if settings.GetString("logFile") != "" {
		f, err := os.OpenFile(settings.GetString("logFile"), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			wd, _ := os.Getwd()
			log.Printf("CWD: %s", wd)
			log.Fatal(err)
		}
		defer f.Close()

		// set output of logs to f
		log.SetOutput(f)
	}

	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	// build features
	var build string
	for _, f := range features {
		build += f + " "
	}
	log.Println("Build tags: " + build)

	// dump them (debugging)
	log.Println("\n>>> settings <<<")
	settings.Dump()
	log.Println("\n>>> settings <<<")

	/*
		Main app
	*/

	// init runtime objects with a real clock
	var runtime = initRuntime(rtc{}, rtc{})

	// start the effect threads so we can update the LEDs
	go runLEDController(settings, runtime)
	go runEffects(settings, runtime)

	// force the LEDs to an on state
	runtime.comms.leds <- ledMessageForce(settings.GetInt("ledAlarm"), modeOn, 0)
	runtime.comms.leds <- ledMessageForce(settings.GetInt("ledError"), modeOn, 0)

	// loader messages?
	if !settings.GetBool("skipLoader") {
		// print the date and time of this build
		showLoader(runtime.comms.effects)
	} else {
		time.Sleep(500 * time.Millisecond)
	}

	// launch the rest of the threads
	go runGetAlarms(settings, runtime)
	go runCheckAlarm(settings, runtime)
	go runWatchButtons(settings, runtime)

	wg.Wait()
}
