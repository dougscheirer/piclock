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
	setupLogging(settings, false)
	// make runtime for test
	return initTestRuntime(settings)
}

func almStateRead(t *testing.T, c chan almStateMsg) (almStateMsg, error) {
	select {
	case e := <-c:
		return e, nil
	default:
		assert.Assert(t, false, "Nothing to read from alarm channel")
	}
	return almStateMsg{}, nil
}

func almStateNoRead(t *testing.T, c chan almStateMsg) (almStateMsg, error) {
	select {
	case e := <-c:
		assert.Assert(t, e == almStateMsg{}, "Got an unexpected value on alarm channel")
	default:
	}
	return almStateMsg{}, nil
}

func ledRead(t *testing.T, c chan ledEffect) (ledEffect, error) {
	select {
	case e := <-c:
		return e, nil
	default:
		assert.Assert(t, false, "Nothing to read from led channel")
	}
	return ledEffect{}, nil
}

func TestCalendarLoadEvents(t *testing.T) {
	runtime := setup()
	clock := runtime.rtc.(clockwork.FakeClock)

	// load alarms
	go runGetAlarms(runtime)

	// block for a sleep
	clock.BlockUntil(1)
	// signal stop and advance clock
	close(runtime.comms.quit)
	clock.Advance(dAlarmSleep)

	// read the chkAlarms comm channel for messages
	state, _ := almStateRead(t, runtime.comms.chkAlarms)
	assert.Assert(t, state.msg == msgLoaded)
	switch v := state.val.(type) {
	case loadedPayload:
		assert.Assert(t, len(v.alarms) == 5)
		assert.Assert(t, v.loadID == 1)
	default:
		assert.Assert(t, false, fmt.Sprintf("Bad value: %v", v))
	}

	// read from the led channel.  running the LED controller
	// might be ideal here, but multi-component testing over time
	// is difficult without refactoring how the run... functions behave
	// to make them fit tests easier.  IMO violates good code over good tests

	// expect 2 led messages, one for turning on the error blink, one to turn it off
	ledBlink, _ := ledRead(t, runtime.comms.leds)
	assert.Assert(t, ledBlink == ledMessage(runtime.settings.GetInt(sLEDErr), modeBlink75, 0))
	ledOff, _ := ledRead(t, runtime.comms.leds)
	assert.Assert(t, ledOff == ledMessage(runtime.settings.GetInt(sLEDErr), modeOff, 0))
}

func TestCalendarLoadEventsFailed(t *testing.T) {
	runtime := setup()
	testEvents := runtime.events.(*testEvents)
	// make it return errors
	testEvents.errorResult = true

	clock := runtime.rtc.(clockwork.FakeClock)

	// load alarms
	go runGetAlarms(runtime)

	// block for a sleep
	clock.BlockUntil(1)
	// signal stop and advance clock
	close(runtime.comms.quit)
	clock.Advance(dAlarmSleep)

	// read the comm channel for (no) messages
	almStateNoRead(t, runtime.comms.chkAlarms)

	// expect 2 led messages, one for turning on the error blink, one to turn it off
	ledBlink, _ := ledRead(t, runtime.comms.leds)
	assert.Assert(t, ledBlink == ledMessage(runtime.settings.GetInt(sLEDErr), modeBlink75, 0))
	ledOff, _ := ledRead(t, runtime.comms.leds)
	assert.Assert(t, ledOff == ledMessage(runtime.settings.GetInt(sLEDErr), modeOff, 0))
}
