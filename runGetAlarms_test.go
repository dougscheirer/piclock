package main

import (
	"fmt"
	"testing"

	"github.com/jonboulle/clockwork"
	"gotest.tools/assert"
)

func TestMain(m *testing.M) {
	piTestMain(m)
}

func TestCalendarLoadEvents(t *testing.T) {
	rt := testRuntime()
	clock := rt.clock.(clockwork.FakeClock)

	// load alarms
	go runGetAlarms(rt)

	// block for 1 sleeps
	clock.BlockUntil(1)
	clock.Advance(dAlarmSleep)
	// signal stop and advance clock
	close(rt.comms.quit)

	// read the chkAlarms comm channel for messages
	state, _ := almStateRead(t, rt.comms.chkAlarms)
	assert.Assert(t, state.msg == msgLoaded)
	switch v := state.val.(type) {
	case loadedPayload:
		assert.Assert(t, len(v.alarms) == 5)
		assert.Assert(t, v.loadID == 1)
	default:
		assert.Assert(t, false, fmt.Sprintf("Bad value: %v", v))
	}

	// expect 2 led messages, one for turning on the error blink, one to turn it off
	ledBlink, _ := ledRead(t, rt.comms.leds)
	assert.Assert(t, ledBlink == ledMessage(rt.settings.GetInt(sLEDErr), modeBlink75, 0))
	ledOff, _ := ledRead(t, rt.comms.leds)
	assert.Assert(t, ledOff == ledMessage(rt.settings.GetInt(sLEDErr), modeOff, 0))
}

func TestCalendarLoadEventsFailed(t *testing.T) {
	rt := testRuntime()
	testEvents := rt.events.(*testEvents)
	// make it return errors
	testEvents.setFails(1)

	clock := rt.clock.(clockwork.FakeClock)

	// load alarms
	go runGetAlarms(rt)

	// block for a sleep
	clock.BlockUntil(1)
	clock.Advance(dAlarmSleep)
	// signal stop and advance clock
	close(rt.comms.quit)
	clock.Advance(dAlarmSleep)

	// read the comm channel for (no) messages
	almStateNoRead(t, rt.comms.chkAlarms)

	// expect 1 led messages, one for turning on the error blink, none to turn it off
	ledBlink, _ := ledRead(t, rt.comms.leds)
	assert.Assert(t, ledBlink == ledMessage(rt.settings.GetInt(sLEDErr), modeBlink75, 0))
	ledNoRead(t, rt.comms.leds)
}

func TestCalendarLoadEventsFailedThenOK(t *testing.T) {
	rt := testRuntime()
	testEvents := rt.events.(*testEvents)
	// make it return errors first
	testEvents.setFails(1)

	clock := rt.clock.(clockwork.FakeClock)

	// load alarms
	go runGetAlarms(rt)

	// block for 1 sleeps
	clock.BlockUntil(1)
	// read the comm channel for (no) messages
	almStateNoRead(t, rt.comms.chkAlarms)
	// expect 1 led messages, one for turning on the error blink, none to turn it off
	ledBlink, _ := ledRead(t, rt.comms.leds)
	assert.Assert(t, ledBlink == ledMessage(rt.settings.GetInt(sLEDErr), modeBlink75, 0))
	ledNoRead(t, rt.comms.leds)

	// advance beyond the refresh time
	clock.Advance(dAlarmSleep)
	clock.Advance(rt.settings.GetDuration(sAlmRefresh))
	clock.BlockUntil(1)
	clock.Advance(dAlarmSleep)
	clock.BlockUntil(1)
	// signal stop and advance clock
	close(rt.comms.quit)
	clock.Advance(dAlarmSleep)

	// now expect that it's fixed
	// read the chkAlarms comm channel for messages
	state, _ := almStateRead(t, rt.comms.chkAlarms)
	assert.Assert(t, state.msg == msgLoaded)
	switch v := state.val.(type) {
	case loadedPayload:
		assert.Assert(t, len(v.alarms) == 5)
		assert.Assert(t, v.loadID == 2)
	default:
		assert.Assert(t, false, fmt.Sprintf("Bad value: %v", v))
	}

	// expect 2 led messages, one for turning on the error blink, one to turn it off
	ledBlink, _ = ledRead(t, rt.comms.leds)
	assert.Assert(t, ledBlink == ledMessage(rt.settings.GetInt(sLEDErr), modeBlink75, 0))
	ledOff, _ := ledRead(t, rt.comms.leds)
	assert.Assert(t, ledOff == ledMessage(rt.settings.GetInt(sLEDErr), modeOff, 0))
}
