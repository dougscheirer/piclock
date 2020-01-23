package main

import (
	"fmt"
	"testing"

	"gotest.tools/assert"

	"github.com/jonboulle/clockwork"
)

func setup() (runtimeConfig, clockwork.FakeClock) {
	// load our test config
	cfgFile := "./test/config.conf"
	settings := initSettings(cfgFile)
	// make runtime for test
	clock := clockwork.NewFakeClock()
	runtime := runtimeConfig{
		settings: settings,
		comms:    initCommChannels(),
		sounds:   noSounds{},
		rtc:      clockwork.NewFakeClock(),
	}

	return runtime, clock
}

func TestCalendarLoadEvents(t *testing.T) {
	runtime, clock := setup()
	// load alarms
	go runGetAlarms(runtime)

	// block for a while?
	clock.BlockUntil(1)

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

	close(runtime.comms.quit)
	clock.BlockUntil(1)
}

func TestCalendarLoadEventsFailed(t *testing.T) {
	runtime, clock := setup()
	// start led controller
	go runLEDController(runtime)

	// load alarms
	go runGetAlarms(runtime)

	// block for a while?
	clock.BlockUntil(1)

	// read from the led channel
}
