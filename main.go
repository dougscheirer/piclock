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

func confirmCalendarAuth(settings configSettings) {
	// this is only for real auth, so specifcally use the gcalEvents impl
	events := &gcalEvents{}
	_, err := events.getCalendarService(settings, true)
	if err == nil {
		// success!
		log.Println("OAuth successful")
		return
	}

	log.Println(err)
}

func main() {
	// CLI args
	args := parseCLIArgs()

	// read config information
	settings := initSettings(args.configFile)

	// are we just generating the oauth token?
	if args.oauth {
		confirmCalendarAuth(settings)
		return
	}

	// first try to set up the log (optional)
	if settings.GetString(sLog) != "" {
		f, err := os.OpenFile(settings.GetString(sLog), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
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
	runtime := initRuntimeConfig(settings)

	// start the effect threads so we can update the LEDs
	go runLEDController(runtime)
	go runEffects(runtime)

	// force the LEDs to an on state
	runtime.comms.leds <- ledMessageForce(settings.GetInt(sLEDAlm), modeOn, 0)
	runtime.comms.leds <- ledMessageForce(settings.GetInt(sLEDErr), modeOn, 0)

	// loader messages?
	if !settings.GetBool(sSkipLoader) {
		// print the date and time of this build
		showLoader(runtime.comms.effects)
	} else {
		runtime.rtc.Sleep(500 * time.Millisecond)
	}

	// launch the rest of the threads
	go runGetAlarms(runtime)
	go runCheckAlarm(runtime)
	go runWatchButtons(runtime)

	wg.Wait()
}
