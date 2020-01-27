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
	assert.Equal(t, es[0].val.(displayPrint).s, "AL:")

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

func TestCheckAlarmsCountdownCancel(t *testing.T) {
	rt, clock, comms := testRuntime()
	events := rt.events.(*testEvents)

	// alarms are set for between 6 and 10, so advance the clock to 5:57.90
	clock.Advance(5*time.Hour + 58*time.Minute)

	go runCheckAlarms(rt)
	// wait for a cycle to complete startup loop
	clock.BlockUntil(1)
	// should have messaged an off
	le, _ := ledRead(t, rt.comms.leds)
	assert.Equal(t, le.pin, rt.settings.GetInt(sLEDAlm))
	assert.Equal(t, le.mode, modeOff)

	// pretend we loaded one alarm
	events.almCount = 1
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
	assert.Equal(t, es[0].val.(displayPrint).s, "AL:")

	// also should have started the countdown
	e, _ := effectRead(t, rt.comms.effects)
	assert.Equal(t, e.id, eCountdown)

	// advance for a bit, but not too far
	testBlockDurationCB(clock, dAlarmSleep, time.Minute, func(int) {
		le, _ = ledRead(t, rt.comms.leds)
	})

	// should *not* have started the alarm effect
	e, _ = effectNoRead(t, rt.comms.effects)

	// cancel with a button press (and release)
	comms.chkAlarms <- mainButtonAlmMsg(true, time.Second)
	testBlockDuration(clock, dAlarmSleep, dAlarmSleep)
	comms.chkAlarms <- mainButtonAlmMsg(false, time.Second)
	testBlockDuration(clock, dAlarmSleep, 2*dAlarmSleep)

	// should see a cancel effect
	e, _ = effectRead(t, rt.comms.effects)
	assert.Equal(t, e.id, eAlarmOff)

	// wait another minute and make sure the alarm did *not* fire
	testBlockDurationCB(clock, dAlarmSleep, time.Minute+time.Second, func(int) {
		le, _ = ledRead(t, rt.comms.leds)
	})

	e, _ = effectNoRead(t, rt.comms.effects)

	// done
	testQuit(rt)
}

func TestCheckAlarmsCountdownMultiCancel(t *testing.T) {
	rt, clock, comms := testRuntime()
	events := rt.events.(*testEvents)

	// alarms are set for between 6 and 10, so advance the clock to 5:57.90
	clock.Advance(5*time.Hour + 58*time.Minute)

	go runCheckAlarms(rt)
	// wait for a cycle to complete startup loop
	clock.BlockUntil(1)
	// should have messaged an off
	le, _ := ledRead(t, rt.comms.leds)
	assert.Equal(t, le.pin, rt.settings.GetInt(sLEDAlm))
	assert.Equal(t, le.mode, modeOff)

	// pretend we loaded two alarms
	events.almCount = 2
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
	assert.Equal(t, es[0].val.(displayPrint).s, "AL:")

	// also should have started the countdown
	e, _ := effectRead(t, rt.comms.effects)
	assert.Equal(t, e.id, eCountdown)

	// advance for a bit, but not too far
	testBlockDurationCB(clock, dAlarmSleep, time.Minute, func(int) {
		le, _ = ledRead(t, rt.comms.leds)
	})

	// should *not* have started the alarm effect
	e, _ = effectNoRead(t, rt.comms.effects)

	// cancel with a button press (and release)
	comms.chkAlarms <- mainButtonAlmMsg(true, time.Second)
	testBlockDuration(clock, dAlarmSleep, dAlarmSleep)
	comms.chkAlarms <- mainButtonAlmMsg(false, time.Second)
	testBlockDuration(clock, dAlarmSleep, 2*dAlarmSleep)

	// should see a cancel effect
	e, _ = effectRead(t, rt.comms.effects)
	assert.Equal(t, e.id, eAlarmOff)

	// wait another minute and make sure the alarm did *not* fire
	testBlockDurationCB(clock, dAlarmSleep, time.Minute+time.Second, func(int) {
		le, _ = ledRead(t, rt.comms.leds)
	})

	// should be the next alarm
	es, _ = effectReads(t, rt.comms.effects, 4)
	assert.Equal(t, es[0].id, ePrint)
	assert.Equal(t, es[0].val.(displayPrint).s, "AL:")

	// done
	testQuit(rt)
}

func TestCheckAlarmsCountdownMultiAlarms(t *testing.T) {
	rt, clock, comms := testRuntime()
	events := rt.events.(*testEvents)

	// alarms are set for between 6 and 10, so advance the clock to 5:57.90
	clock.Advance(5*time.Hour + 58*time.Minute)

	go runCheckAlarms(rt)
	// wait for a cycle to complete startup loop
	clock.BlockUntil(1)
	// should have messaged an off
	le, _ := ledRead(t, rt.comms.leds)
	assert.Equal(t, le.pin, rt.settings.GetInt(sLEDAlm))
	assert.Equal(t, le.mode, modeOff)

	// pretend we loaded three alarms
	events.almCount = 3
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
	assert.Equal(t, es[0].val.(displayPrint).s, "AL:")

	// also should have started the countdown
	e, _ := effectRead(t, rt.comms.effects)
	assert.Equal(t, e.id, eCountdown)

	// advance through the next alarm
	testBlockDurationCB(clock, dAlarmSleep, 2*time.Minute+time.Second, func(int) {
		le, _ = ledRead(t, rt.comms.leds)
	})

	// should have started the alarm effect
	e, _ = effectRead(t, rt.comms.effects)
	assert.Equal(t, e.id, eAlarmOn)

	// signal a done and wait
	rt.sounds.(*noSounds).done <- true
	testBlockDuration(clock, dAlarmSleep, dAlarmSleep)

	// should get another alarm notice
	es, _ = effectReads(t, rt.comms.effects, 4)
	assert.Equal(t, es[0].id, ePrint)
	assert.Equal(t, es[0].val.(displayPrint).s, "AL:")

	// wait another minute and make sure the alarm did *not* fire
	testBlockDurationCB(clock, dAlarmSleep, time.Minute+time.Second, func(int) {
		le, _ = ledRead(t, rt.comms.leds)
	})

	// should be the next alarm
	es, _ = effectReads(t, rt.comms.effects, 4)
	assert.Equal(t, es[0].id, ePrint)
	assert.Equal(t, es[0].val.(displayPrint).s, "AL:")

	// TODO: wait for a second alarm in another hour

	// wait for it to end

	// one more alarm to go

	// done
	testQuit(rt)
}
