package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// piclock -config={config file}

var wg sync.WaitGroup

var features = []string{}

func confirmCalendarAuth(settings configSettings) {
	// this is only for real auth, so specifcally use the gcalEvents impl
	events := &gcalEvents{}
	rt := runtimeConfig{
		settings: settings,
		logger:   &ThreadLogger{name: "confirmCalendarAuth"},
	}
	_, err := events.getCalendarService(rt, true)
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

	// are we just showing the version?
	if args.version {
		showVersionInfo()
		return
	}

	// read config information
	settings := initSettings(args.configFile)

	// are we just generating the oauth token?
	if args.oauth {
		confirmCalendarAuth(settings)
		return
	}

	// first try to set up the log (optional)
	setupLogging(settings, true)

	log.Printf("STARTUP : %d\n", os.Getpid())
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

	// start the sigterm thread
	startWaitSigterm(rt)

	// start the effect threads so we can update the LEDs
	startLEDController(rt)
	startEffects(rt)

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
	startGetAlarms(rt)
	startCheckAlarms(rt)
	startWatchButtons(rt)
	// optional config service
	if settings.GetInt(sConfigSvc) > 0 {
		startConfigService(rt)
	}

	wg.Wait()
}

func startWaitSigterm(rt runtimeConfig) {
	wg.Add(1)
	go runWaitSigterm(rt)
}

func runWaitSigterm(rt runtimeConfig) {
	defer wg.Done()

	// make a signal channel to listen to
	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	select {
	case <-sigChan:
		// signal a stop
		fmt.Print("exiting startWaitSigterm")
		// close(rt.comms.quit)
	}
}
