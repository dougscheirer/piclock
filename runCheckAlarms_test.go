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
	comms.chkAlarms <- mainButtonAlmMsg(true, 0)
	testBlockDuration(clock, dAlarmSleep, dAlarmSleep)
	comms.chkAlarms <- mainButtonAlmMsg(false, 0)
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
	comms.chkAlarms <- mainButtonAlmMsg(true, 0)
	testBlockDuration(clock, dAlarmSleep, dAlarmSleep)
	comms.chkAlarms <- mainButtonAlmMsg(false, 0)
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

	// should get another alarm notice
	es, _ = effectReads(t, rt.comms.effects, 4)
	assert.Equal(t, es[0].id, ePrint)
	assert.Equal(t, es[0].val.(displayPrint).s, "AL:")

	// wait another minute and make sure the alarm did *not* fire
	testBlockDurationCB(clock, dAlarmSleep, time.Minute+time.Second, func(int) {
		le, _ = ledRead(t, rt.comms.leds)
	})

	// wait for a second alarm in another hour
	testBlockDurationCB(clock, dAlarmSleep, 60*time.Minute, func(int) {
		le, _ = ledRead(t, rt.comms.leds)
	})

	// also should have started the countdown
	e, _ = effectRead(t, rt.comms.effects)
	assert.Equal(t, e.id, eCountdown)

	// should also be an alarm effect
	e, _ = effectRead(t, rt.comms.effects)
	assert.Equal(t, e.id, eAlarmOn)

	// should get another alarm notice
	es, _ = effectReads(t, rt.comms.effects, 4)
	assert.Equal(t, es[0].id, ePrint)
	assert.Equal(t, es[0].val.(displayPrint).s, "AL:")

	// done
	testQuit(rt)
}

func TestCheckAlarmsReloadButton(t *testing.T) {
	rt, clock, comms := testRuntime()
	// events := rt.events.(*testEvents)

	go runCheckAlarms(rt)
	// wait for a cycle to complete startup loop
	clock.BlockUntil(1)

	// the long hold comes in on another message
	comms.chkAlarms <- longButtonAlmMsg(true)
	testBlockDuration(clock, dAlarmSleep, dAlarmSleep)

	// should have sent a reload message to getAlarms
	e, _ := almStateRead(t, rt.comms.getAlarms)
	assert.Equal(t, e.ID, msgReload)

	testQuit(rt)
}

func TestCheckAlarmsDoubleClick(t *testing.T) {
	rt, clock, comms := testRuntime()
	// events := rt.events.(*testEvents)

	go runCheckAlarms(rt)
	// wait for a cycle to complete startup loop
	clock.BlockUntil(1)

	// the double click event comes in on another message
	comms.chkAlarms <- doubleButtonAlmMsg(true)
	testBlockDuration(clock, dAlarmSleep, dAlarmSleep)

	// should have sent effects
	d, _ := effectRead(t, comms.effects)
	assert.Equal(t, d.id, ePrint)
	assert.Equal(t, d.val.(displayPrint).s, "none")

	testQuit(rt)
}

func TestCheckAlarmsDoubleClickPending(t *testing.T) {
	rt, clock, comms := testRuntime()
	events := rt.events.(*testEvents)
	// pretend we loaded some alarms, all or old
	events.oldAlarms = 3
	alarms, _ := getAlarmsFromService(rt)
	comms.chkAlarms <- alarmsLoadedMsg(1, alarms, true)

	go runCheckAlarms(rt)
	// wait for a cycle to complete startup loop
	clock.BlockUntil(1)

	// clear the effects channel of all the loading stuff
	effectReadAll(comms.effects)

	// the double click event comes in on another message
	comms.chkAlarms <- doubleButtonAlmMsg(true)
	testBlockDuration(clock, dAlarmSleep, dAlarmSleep)

	// should have sent effects
	de, _ := effectReads(t, comms.effects, 4)

	compares := []string{"AL:", "09:00", "01.26", "2020"}
	for i := range compares {
		assert.Equal(t, de[i].id, ePrint)
		assert.Equal(t, de[i].val.(displayPrint).s, compares[i])
	}
}
func TestCheckAlarmsReloadButtonAlarmsAtStart(t *testing.T) {
	rt, clock, comms := testRuntime()
	events := rt.events.(*testEvents)

	// pretend we loaded some alarms
	events.almCount = 3
	alarms, _ := getAlarmsFromService(rt)
	comms.chkAlarms <- alarmsLoadedMsg(1, alarms, true)

	go runCheckAlarms(rt)
	// wait for a cycle to complete startup loop
	clock.BlockUntil(1)

	// send the long press message
	comms.chkAlarms <- longButtonAlmMsg(true)
	testBlockDuration(clock, dAlarmSleep, 3*dAlarmSleep)
	comms.chkAlarms <- longButtonAlmMsg(false)

	// should have sent a reload message to getAlarms
	e, _ := almStateRead(t, rt.comms.getAlarms)
	assert.Equal(t, e.ID, msgReload)

	// pretend like we reloaded new alarms
	alarms, _ = getAlarmsFromService(rt)
	comms.chkAlarms <- alarmsLoadedMsg(1, alarms, true)

	// expect a report on the next alarm since we explicitly reloaded
	testBlockDuration(clock, dAlarmSleep, 5*time.Second)

	// read from the effects
	de, _ := effectReads(t, comms.effects, 4)
	assert.Equal(t, len(de), 4)
	compares := []string{"AL:", "06:00", "01.26", "2020"}
	for i := range compares {
		assert.Equal(t, de[i].id, ePrint)
		assert.Equal(t, de[i].val.(displayPrint).s, compares[i])
	}

	testQuit(rt)
}

func TestCheckAlarmsFiredCancel(t *testing.T) {
	rt, clock, comms := testRuntime()
	rt.settings.settings[sCountdown] = time.Duration(0)

	events := rt.events.(*testEvents)

	// alarms are set for between 6 and 10, so advance the clock to 5:59.30
	clock.Advance(5*time.Hour + 59*time.Minute + 30*time.Second)

	go runCheckAlarms(rt)
	// wait for a cycle to complete startup loop
	clock.BlockUntil(1)
	// should have messaged an off
	le, _ := ledRead(t, rt.comms.leds)
	assert.Equal(t, le.pin, rt.settings.GetInt(sLEDAlm))
	assert.Equal(t, le.mode, modeOff)

	// pretend we loaded one alarm
	events.almCount = 4
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

	// should not have started countdown
	effectNoRead(t, rt.comms.effects)

	// advance until after the alarm starts
	testBlockDurationCB(clock, dAlarmSleep, 2*time.Minute, func(int) {
		le, _ = ledRead(t, rt.comms.leds)
	})

	// should see a play signal and bunch of prints that will get queued
	e, _ := effectRead(t, rt.comms.effects)
	assert.Equal(t, e.id, eAlarmOn)
	es, _ = effectReads(t, rt.comms.effects, 4)
	compares := []string{"AL:", "07:00", "01.26", "2020"}
	for i := range compares {
		assert.Equal(t, es[i].id, ePrint)
		assert.Equal(t, es[i].val.(displayPrint).s, compares[i])
	}

	// cancel with a button press (and release)
	comms.chkAlarms <- mainButtonAlmMsg(true, 0)
	testBlockDuration(clock, dAlarmSleep, dAlarmSleep)
	comms.chkAlarms <- mainButtonAlmMsg(false, 0)
	testBlockDuration(clock, dAlarmSleep, 2*dAlarmSleep)

	// should see the cancel effect
	e, _ = effectRead(t, rt.comms.effects)
	assert.Equal(t, e.id, eAlarmOff)

	// wait another minute and make sure nothing else happens
	testBlockDurationCB(clock, dAlarmSleep, time.Minute+time.Second, func(int) {
		le, _ = ledRead(t, rt.comms.leds)
	})

	e, _ = effectNoRead(t, rt.comms.effects)

	// TODO:advance to the next alarm and retest

	// done
	testQuit(rt)
}
