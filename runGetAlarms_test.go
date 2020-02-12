package main

import (
	"fmt"
	"testing"

	"gotest.tools/assert"
)

func TestCalendarLoadEvents(t *testing.T) {
	rt, clock, comms := testRuntime()

	// load alarms
	go runGetAlarms(rt)

	// block for 1 sleeps
	clock.BlockUntil(1)
	clock.Advance(2 * dAlarmSleep)
	// signal stop
	close(rt.comms.quit)
	clock.Advance(dAlarmSleep)

	// read the chkAlarms comm channel for messages
	states := almStateReadAll(comms.chkAlarms)
	// should be a non-error config msg and a loaded msg
	assert.Equal(t, len(states), 2)
	assert.Equal(t, states[0].ID, msgConfigError)
	assert.Equal(t, states[0].val.(configError).err, false)

	switch v := states[1].val.(type) {
	case loadedPayload:
		assert.Equal(t, len(v.alarms), 5)
		assert.Equal(t, v.loadID, 1)
	default:
		assert.Equal(t, false, fmt.Sprintf("Bad value: %v", v))
	}

	// expect 2 led messages, one for turning on the error blink, one to turn it off
	ledBlink, _ := ledRead(t, comms.leds)
	assert.Equal(t, ledBlink, ledMessage(rt.settings.GetInt(sLEDErr), modeBlink75, 0))
	ledOff, _ := ledRead(t, comms.leds)
	assert.Equal(t, ledOff, ledMessage(rt.settings.GetInt(sLEDErr), modeOff, 0))

	// done
	testQuit(rt)
}

func TestCalendarLoadEventsFailed(t *testing.T) {
	rt, clock, comms := testRuntime()
	testEvents := rt.events.(*testEvents)
	// make it return errors
	testEvents.setFails(1)

	// load alarms
	go runGetAlarms(rt)

	// block for a sleep
	clock.BlockUntil(1)
	clock.Advance(dAlarmSleep)
	// signal stop and advance clock
	close(comms.quit)
	clock.Advance(dAlarmSleep)

	// read the comm channel for messages
	aA := almStateReadAll(comms.chkAlarms)
	assert.Equal(t, len(aA), 1)
	assert.Equal(t, aA[0].ID, msgConfigError)
	assert.Equal(t, aA[0].val.(configError).err, true)
	assert.Equal(t, aA[0].val.(configError).secret, "0001")

	// expect 1 led messages, one for turning on the error blink, none to turn it off
	ledBlink, _ := ledRead(t, comms.leds)
	assert.Equal(t, ledBlink, ledMessage(rt.settings.GetInt(sLEDErr), modeBlink75, 0))
	lA := ledReadAll(comms.leds)
	assert.Equal(t, len(lA), 0)

	// done
	testQuit(rt)
}

func TestCalendarLoadEventsFailedThenOK(t *testing.T) {
	rt, clock, comms := testRuntime()
	testEvents := rt.events.(*testEvents)
	// make it return errors first
	testEvents.setFails(1)

	// load alarms
	go runGetAlarms(rt)

	// block for 1 sleeps
	clock.BlockUntil(1)
	// read the comm channel for error message
	aA := almStateReadAll(comms.chkAlarms)
	assert.Equal(t, len(aA), 1)
	assert.Equal(t, aA[0].ID, msgConfigError)
	assert.Equal(t, aA[0].val.(configError).err, true)
	assert.Equal(t, aA[0].val.(configError).secret, "0001")

	// expect 1 led messages, one for turning on the error blink, none to turn it off
	ledBlink, _ := ledRead(t, comms.leds)
	assert.Equal(t, ledBlink, ledMessage(rt.settings.GetInt(sLEDErr), modeBlink75, 0))
	lA := ledReadAll(comms.leds)
	assert.Equal(t, len(lA), 0)

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
	states := almStateReadAll(rt.comms.chkAlarms)
	assert.Equal(t, len(states), 2)
	assert.Equal(t, states[0].ID, msgConfigError)
	assert.Equal(t, states[0].val.(configError).err, false)
	assert.Equal(t, states[0].val.(configError).secret, "0002")

	assert.Equal(t, states[1].ID, msgLoaded)
	switch v := states[1].val.(type) {
	case loadedPayload:
		assert.Equal(t, len(v.alarms), 5)
		assert.Equal(t, v.loadID, 2)
	default:
		assert.Equal(t, false, fmt.Sprintf("Bad value: %v", v))
	}

	// expect 2 led messages, one for turning on the error blink, one to turn it off
	ledBlink, _ = ledRead(t, rt.comms.leds)
	assert.Equal(t, ledBlink, ledMessage(rt.settings.GetInt(sLEDErr), modeBlink75, 0))
	ledOff, _ := ledRead(t, rt.comms.leds)
	assert.Equal(t, ledOff, ledMessage(rt.settings.GetInt(sLEDErr), modeOff, 0))

	// done
	testQuit(rt)
}

func TestCalendarLoadOldEvents(t *testing.T) {
	rt, clock, comms := testRuntime()
	events := rt.events.(*testEvents)
	events.oldAlarms = 2 // skip to old alarms

	// load alarms
	go runGetAlarms(rt)

	// block for 1 sleeps
	clock.BlockUntil(1)
	clock.Advance(dAlarmSleep)
	// signal stop
	close(rt.comms.quit)
	clock.Advance(dAlarmSleep)

	// read the chkAlarms comm channel for messages
	states := almStateReadAll(comms.chkAlarms)
	assert.Equal(t, len(states), 2)
	assert.Equal(t, states[0].ID, msgConfigError)
	assert.Equal(t, states[0].val.(configError).err, false)

	assert.Equal(t, states[1].ID, msgLoaded)
	switch v := states[1].val.(type) {
	case loadedPayload:
		assert.Equal(t, len(v.alarms), 3)
		assert.Equal(t, v.loadID, 1)
	default:
		assert.Equal(t, false, fmt.Sprintf("Bad value: %v", v))
	}

	// expect 2 led messages, one for turning on the error blink, one to turn it off
	ledBlink, _ := ledRead(t, comms.leds)
	assert.Equal(t, ledBlink, ledMessage(rt.settings.GetInt(sLEDErr), modeBlink75, 0))
	ledOff, _ := ledRead(t, comms.leds)
	assert.Equal(t, ledOff, ledMessage(rt.settings.GetInt(sLEDErr), modeOff, 0))

	// done
	testQuit(rt)
}
