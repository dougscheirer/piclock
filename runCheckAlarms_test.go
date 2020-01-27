package main

import (
	"testing"
	"time"

	"gotest.tools/assert"
)

/* things that runCheckAlarms does:

listens for almState messages (either 'loaded' or 'main button')
walks the alarm list looking for an upcoming alarm
notifies effects about countdown, alarm start, alarm stop
turns on alarm led is a pending alarm is found

*/

func TestCheckAlarmsNone(t *testing.T) {
	rt, clock, comms := testRuntime()

	go runCheckAlarms(rt)

	// pretend we loaded some alarms, all or old
	alarms := []alarm{}
	comms.chkAlarms <- alarmsLoadedMsg(1, alarms, true)

	// wait for a cycle
	testBlockDuration(clock, dAlarmSleep, dAlarmSleep)

	// should have messaged an off to the led controller
	le, _ := ledRead(t, rt.comms.leds)
	assert.Equal(t, le.pin, rt.settings.GetInt(sLEDAlm))
	assert.Equal(t, le.mode, modeOff)
	// done
	testQuit(rt)
}

func TestCheckAlarmsOld(t *testing.T) {
	rt, clock, comms := testRuntime()
	events := rt.events.(*testEvents)

	go runCheckAlarms(rt)

	// pretend we loaded some alarms, all or old
	events.oldAlarms = 5
	alarms, _ := getAlarmsFromService(rt)
	comms.chkAlarms <- alarmsLoadedMsg(1, alarms, true)

	// wait for a cycle
	testBlockDuration(clock, dAlarmSleep, dAlarmSleep)

	// should have messaged an off to the led controller
	le, _ := ledRead(t, rt.comms.leds)
	assert.Equal(t, le.pin, rt.settings.GetInt(sLEDAlm))
	assert.Equal(t, le.mode, modeOff)
	// done
	testQuit(rt)
}

func TestCheckAlarmsMixed(t *testing.T) {
	rt, clock, comms := testRuntime()
	events := rt.events.(*testEvents)

	go runCheckAlarms(rt)
	// wait for a cycle to complete startup loop
	clock.BlockUntil(1)
	// should have messaged an off
	le, _ := ledRead(t, rt.comms.leds)
	assert.Equal(t, le.pin, rt.settings.GetInt(sLEDAlm))
	assert.Equal(t, le.mode, modeOff)

	// pretend we loaded some alarms, all or old
	events.oldAlarms = 3
	alarms, _ := getAlarmsFromService(rt)
	comms.chkAlarms <- alarmsLoadedMsg(1, alarms, true)

	// wait for a cycle
	testBlockDuration(clock, dAlarmSleep, dAlarmSleep)

	// should have gotten an on
	le, _ = ledRead(t, rt.comms.leds)
	assert.Equal(t, le.pin, rt.settings.GetInt(sLEDAlm))
	assert.Equal(t, le.mode, modeOn)
	// done
	testQuit(rt)
}

func TestCheckAlarmsCountdown(t *testing.T) {
	rt, clock, comms := testRuntime()

	// alarms are set for between 6 and 10, so advance the clock to 5:57.90
	clock.Advance(5*time.Hour + 58*time.Minute)

	go runCheckAlarms(rt)
	// wait for a cycle to complete startup loop
	clock.BlockUntil(1)
	// should have messaged an off
	le, _ := ledRead(t, rt.comms.leds)
	assert.Equal(t, le.pin, rt.settings.GetInt(sLEDAlm))
	assert.Equal(t, le.mode, modeOff)

	// pretend we loaded some alarms
	alarms, _ := getAlarmsFromService(rt)
	comms.chkAlarms <- alarmsLoadedMsg(1, alarms, true)

	// wait for a cycle
	testBlockDuration(clock, dAlarmSleep, dAlarmSleep)

	// should have gotten an on
	le, _ = ledRead(t, rt.comms.leds)
	assert.Equal(t, le.pin, rt.settings.GetInt(sLEDAlm))
	assert.Equal(t, le.mode, modeOn)

	// should have gotten a bunch of prints
	es, _ := effectReads(t, rt.comms.effects, 4)
	assert.Equal(t, es[0].id, ePrint)
	assert.Equal(t, es[0].val.(displayPrint).s, "AL:1")

	// also should have started the countdown
	e, _ := effectRead(t, rt.comms.effects)
	assert.Equal(t, e.id, eCountdown)

	// advance util after the alarm starts
	// make sure to read from the led channel or the test will block
	testBlockDurationCB(clock, dAlarmSleep, 2*time.Minute+dAlarmSleep, func(int) {
		le, _ = ledRead(t, rt.comms.leds)
	})

	// should have started the alarm effect
	e, _ = effectRead(t, rt.comms.effects)
	assert.Equal(t, e.id, eAlarmOn)

	// done
	testQuit(rt)
}
