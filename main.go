package main

import (
	"log"
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
	f, _ := setupLogging(settings, true)
	if f != nil {
		defer f.Close()
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

	// init rt objects with a real clock
	rt := initRuntimeConfig(settings)

	// start the effect threads so we can update the LEDs
	go runLEDController(rt)
	go runEffects(rt)

	// force the LEDs to an on state
	rt.comms.leds <- ledMessageForce(settings.GetInt(sLEDAlm), modeOn, 0)
	rt.comms.leds <- ledMessageForce(settings.GetInt(sLEDErr), modeOn, 0)

	// loader messages?
	if !settings.GetBool(sSkipLoader) {
		// print the date and time of this build
		showLoader(rt)
	} else {
		rt.clock.Sleep(250 * time.Millisecond)
	}

	// launch the rest of the threads
	go runGetAlarms(rt)
	go runCheckAlarms(rt)
	go runWatchButtons(rt)
	go runConfigService(rt)

	wg.Wait()
}
