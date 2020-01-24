package main

import (
	"fmt"
	"testing"

	"github.com/jonboulle/clockwork"
	"gotest.tools/assert"
)

func setup() runtimeConfig {
	// load our test config
	cfgFile := "./test/config.conf"
	settings := initSettings(cfgFile)
	// make runtime for test
	return initTestRuntime(settings)
}

func TestCalendarLoadEvents(t *testing.T) {
	runtime := setup()
	clock := runtime.rtc.(clockwork.FakeClock)

	// load alarms
	go runGetAlarms(runtime)

	// block for a while?
	clock.BlockUntil(1)
	// signal stop and advance clock
	close(runtime.comms.quit)
	clock.Advance(dAlarmSleep)

	// read the comm channel for messages
	state := <-runtime.comms.almState
	assert.Assert(t, state.msg == msgLoaded)
	switch v := state.val.(type) {
	case loadedPayload:
		assert.Assert(t, len(v.alarms) == 5)
		assert.Assert(t, v.loadID == 1)
	default:
		assert.Assert(t, false, fmt.Sprintf("Bad value: %v", v))
	}

	// make a new quit channel for the new thread
	runtime.comms.quit = make(chan struct{}, 1)
	go runLEDController(runtime)
	clock.BlockUntil(1)

	// read from the led channel (or check the led stub)
	var logger *logLed = runtime.led.(*logLed)
	assert.Assert(t, logger.leds[runtime.settings.GetInt(sLEDAlm)])
	close(runtime.comms.quit)
}

func TestCalendarLoadEventsFailed(t *testing.T) {
	runtime := setup()
	clock := runtime.rtc.(clockwork.FakeClock)

	// load alarms
	go runGetAlarms(runtime)

	// block for a while?
	clock.BlockUntil(1)

	//
}
