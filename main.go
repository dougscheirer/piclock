package main

import (
	"fmt"
	"sync"
)

var wg sync.WaitGroup

// piclock -config={config file}

func main() {
	// read config information
	settings := InitSettings()

	// dump them (debugging)
	fmt.Println("\n>>> Settings <<<\n")
	settings.Dump()
	fmt.Println("\n>>> Settings <<<\n")

	/*
		Main app
	*/

	// TODO: move these into a struct?
  alarmChannel := make(chan Alarm, 1)
  effectChannel := make(chan Effect, 1)
  handledChannel := make(chan Alarm, 1)

	// wait on our three workers: alarm fetcher, clock runner, alarm checker, button checker
  wg.Add(4)

  // start the effect thread so we can update the LEDs
	go runEffects(settings, effectChannel)

	// google calendar requires OAuth access, so make sure we get it
	// before we go into the main loop
	confirm_calendar_auth(settings, effectChannel)

	go getAlarms(settings, alarmChannel, effectChannel, handledChannel)
	go checkAlarm(settings, alarmChannel, effectChannel, handledChannel)
	go watchButtons(settings, effectChannel)

	wg.Wait()
}
