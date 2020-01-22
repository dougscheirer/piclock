package main

import "testing"

func TestCalendar(t *testing.T) {
	// load our test config
	cfgFile := "./test/config.conf"
	settings := initSettings(cfgFile)
	// make runtime for test
	runtime := initRuntime(testClock{})

	// launch some threads
	go runEffects(settings, runtime)
	go runLEDController(settings, runtime)

	// load alarms

	// now advance time to trigger alarm

	// see if the alarm triggered

	t.Error("Failed!")
}
