package main

import (
	"sync"
	"log"
	"os"
)

// piclock -config={config file}

var wg sync.WaitGroup

func main() {
	// read config information
	settings := InitSettings()

	// first try to set up the log
	f, err := os.OpenFile(settings.GetString("logFile"), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
  if err != nil {
    log.Fatal(err)
  }

  //defer to close when you're done with it, not because you think it's idiomatic!
  defer f.Close()

  //set output of logs to f
  log.SetOutput(f)
  log.SetFlags(log.Ldate|log.Ltime|log.Lmicroseconds)

	// dump them (debugging)
	log.Println("\n>>> Settings <<<\n")
	settings.Dump()
	log.Println("\n>>> Settings <<<\n")

	/*
		Main app
	*/

	// TODO: move these into a struct?
  alarmChannel := make(chan CheckMsg, 1)
  effectChannel := make(chan Effect, 1)
  loaderChannel := make(chan LoaderMsg, 1)

	// wait on our workers:
	// alarm fetcher
	// clock runner
	// alarm checker
	// button checker
  wg.Add(4)

  // start the effect thread so we can update the LEDs
	go runEffects(settings, effectChannel, loaderChannel)

	// google calendar requires OAuth access, so make sure we get it
	// before we go into the main loop
	confirm_calendar_auth(settings, effectChannel)

	go getAlarms(settings, alarmChannel, effectChannel, loaderChannel)
	go checkAlarm(settings, alarmChannel, effectChannel, loaderChannel)
	go watchButtons(settings, effectChannel)

	wg.Wait()
}
