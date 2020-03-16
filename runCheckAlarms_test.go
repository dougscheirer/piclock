package main

import (
	"sort"
	"testing"
	"time"

	"github.com/jonboulle/clockwork"
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

	// alarms are set for between 6 and 10, so advance the clock to 5:58
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
	es := effectReadAll(rt.comms.effects)
	assert.Equal(t, len(es), 3) // next Al... 0:01 (countdown)
	assert.Equal(t, es[0].id, ePrintRolling)
	assert.Equal(t, es[0].val.(displayPrint).s, sNextALIn)
	assert.Equal(t, es[1].val.(displayPrint).s, " 0:01")
	// also should have started the countdown
	assert.Equal(t, es[2].id, eCountdown)

	// advance util after the alarm starts
	// make sure to read from the led channel or the test will block
	testBlockDurationCB(clock, dAlarmSleep, 2*time.Minute+dAlarmSleep, func(int) {
		le, _ = ledRead(t, rt.comms.leds)
	})

	// should have started the alarm effect
	e, _ := effectRead(t, rt.comms.effects)
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

	es := effectReadAll(rt.comms.effects)
	assert.Equal(t, len(es), 3) // next Al... 0:01 (countdown)
	assert.Equal(t, es[0].id, ePrintRolling)
	assert.Equal(t, es[0].val.(displayPrint).s, sNextALIn)
	assert.Equal(t, es[1].val.(displayPrint).s, " 0:01")
	// also should have started the countdown
	assert.Equal(t, es[2].id, eCountdown)

	// advance for a bit, but not too far
	testBlockDurationCB(clock, dAlarmSleep, time.Minute, func(count int) {
		le, _ = ledRead(t, rt.comms.leds)
	})

	// should *not* have started the alarm effect
	eA := effectReadAll(rt.comms.effects)
	assert.Equal(t, len(eA), 0)

	// cancel with a button press (and release)
	comms.chkAlarms <- mainButtonAlmMsg(true, 0)
	testBlockDuration(clock, dAlarmSleep, dAlarmSleep)
	comms.chkAlarms <- mainButtonAlmMsg(false, 0)
	testBlockDuration(clock, dAlarmSleep, 2*dAlarmSleep)

	// should see a cancel effect
	eA = effectReadAll(rt.comms.effects)
	assert.Equal(t, eA[0].id, eAlarmOff)
	assert.Equal(t, eA[1].id, ePrint)
	assert.Equal(t, eA[1].val.(displayPrint).s, "none")
	assert.Equal(t, len(eA), 2)

	// wait another minute and make sure the alarm did *not* fire
	testBlockDurationCB(clock, dAlarmSleep, time.Minute+time.Second, func(int) {
		le, _ = ledRead(t, rt.comms.leds)
	})

	// should have no effects
	eA = effectReadAll(rt.comms.effects)
	assert.Equal(t, len(eA), 0)

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

	es := effectReadAll(rt.comms.effects)
	assert.Equal(t, len(es), 3) // next Al... 0:01 (countdown)
	assert.Equal(t, es[0].id, ePrintRolling)
	assert.Equal(t, es[0].val.(displayPrint).s, sNextALIn)
	assert.Equal(t, es[1].val.(displayPrint).s, " 0:01")
	// also should have started the countdown
	assert.Equal(t, es[2].id, eCountdown)

	// advance for a bit, but not too far
	testBlockDurationCB(clock, dAlarmSleep, time.Minute, func(int) {
		le, _ = ledRead(t, rt.comms.leds)
	})

	// should *not* have started the alarm effect
	eA := effectReadAll(rt.comms.effects)
	assert.Equal(t, len(eA), 0)

	// cancel with a button press (and release)
	comms.chkAlarms <- mainButtonAlmMsg(true, 0)
	testBlockDuration(clock, dAlarmSleep, dAlarmSleep)
	comms.chkAlarms <- mainButtonAlmMsg(false, 0)
	testBlockDuration(clock, dAlarmSleep, 2*dAlarmSleep)

	// should see a cancel effect
	e, _ := effectRead(t, rt.comms.effects)
	assert.Equal(t, e.id, eAlarmOff)

	// wait another minute and make sure the alarm did *not* fire
	testBlockDurationCB(clock, dAlarmSleep, time.Minute+time.Second, func(int) {
		le, _ = ledRead(t, rt.comms.leds)
	})

	// should be the next alarm
	es = effectReadAll(rt.comms.effects)
	assert.Equal(t, len(es), 2) // next Al... 1:00
	assert.Equal(t, es[0].id, ePrintRolling)
	assert.Equal(t, es[0].val.(displayPrint).s, sNextALIn)
	assert.Equal(t, es[1].val.(displayPrint).s, " 1:00")

	// done
	testQuit(rt)
}

func TestCheckAlarmsCountdownSingle(t *testing.T) {
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
	es := effectReadAll(rt.comms.effects)
	assert.Equal(t, len(es), 3) // next Al... 0:01 (countdown)
	assert.Equal(t, es[0].id, ePrintRolling)
	assert.Equal(t, es[0].val.(displayPrint).s, sNextALIn)
	assert.Equal(t, es[1].val.(displayPrint).s, " 0:01")
	// also should have started the countdown
	assert.Equal(t, es[2].id, eCountdown)

	// advance through the next alarm
	testBlockDurationCB(clock, dAlarmSleep, 120*time.Second, func(int) {
		le, _ = ledRead(t, rt.comms.leds)
	})

	// should have started the alarm effect
	e, _ := effectRead(t, rt.comms.effects)
	assert.Equal(t, e.id, eAlarmOn)

	// should not have gotten a bunch of prints
	es = effectReadAll(rt.comms.effects)
	assert.Equal(t, len(es), 0)

	// cancel the alarm firing, wait for some cycles to process
	comms.chkAlarms <- mainButtonAlmMsg(true, 0)
	testBlockDuration(clock, dAlarmSleep, 10*dAlarmSleep)

	// should have gotten the cancel
	e, _ = effectRead(t, rt.comms.effects)
	assert.Equal(t, e.id, eAlarmOff)

	// also the next alarm msg
	es = effectReadAll(rt.comms.effects)
	assert.Equal(t, len(es), 1) // none
	assert.Equal(t, es[0].id, ePrint)
	assert.Equal(t, es[0].val.(displayPrint).s, "none")

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
	es := effectReadAll(rt.comms.effects)
	assert.Equal(t, len(es), 3) // next Al... 0:01 (countdown)
	assert.Equal(t, es[0].id, ePrintRolling)
	assert.Equal(t, es[0].val.(displayPrint).s, sNextALIn)
	assert.Equal(t, es[1].val.(displayPrint).s, " 0:01")
	// also should have started the countdown
	assert.Equal(t, es[2].id, eCountdown)

	// advance through the next alarm
	testBlockDurationCB(clock, dAlarmSleep, 2*time.Minute+time.Second, func(int) {
		le, _ = ledRead(t, rt.comms.leds)
	})

	// should have started the alarm effect
	e, _ := effectRead(t, rt.comms.effects)
	assert.Equal(t, e.id, eAlarmOn)

	// should not have gotten a bunch of prints
	es = effectReadAll(rt.comms.effects)
	assert.Equal(t, len(es), 0)

	// cancel the alarm firing, wait for a cycle to process
	comms.chkAlarms <- mainButtonAlmMsg(true, 0)
	testBlockDuration(clock, dAlarmSleep, dAlarmSleep)

	// should have gotten the cancel
	e, _ = effectRead(t, rt.comms.effects)
	assert.Equal(t, e.id, eAlarmOff)

	// also the next alarm msg
	es = effectReadAll(rt.comms.effects)
	assert.Equal(t, len(es), 2) // next Al... 0:59
	assert.Equal(t, es[0].id, ePrintRolling)
	assert.Equal(t, es[0].val.(displayPrint).s, sNextALIn)
	assert.Equal(t, es[1].val.(displayPrint).s, " 0:59")

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

	// cancel the alarm firing, wait for a cycle to process
	comms.chkAlarms <- mainButtonAlmMsg(true, 0)
	testBlockDuration(clock, dAlarmSleep, dAlarmSleep)

	// should have gotten the cancel
	e, _ = effectRead(t, rt.comms.effects)
	assert.Equal(t, e.id, eAlarmOff)

	// also the next alarm msg
	es = effectReadAll(rt.comms.effects)
	assert.Equal(t, len(es), 2) // next Al... 0:58
	assert.Equal(t, es[0].id, ePrintRolling)
	assert.Equal(t, es[0].val.(displayPrint).s, sNextALIn)
	assert.Equal(t, es[1].val.(displayPrint).s, " 0:58")

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
	comms.chkAlarms <- longButtonAlmMsg(true, 0)
	testBlockDuration(clock, dAlarmSleep, dAlarmSleep)
	comms.chkAlarms <- longButtonAlmMsg(false, time.Second)
	testBlockDuration(clock, dAlarmSleep, dAlarmSleep)

	// should have sent a reload message to getAlarms
	e, _ := almStateRead(t, rt.comms.getAlarms)
	assert.Equal(t, e.ID, msgReload)

	// try it again, just to make sure that the state reset properly
	comms.chkAlarms <- longButtonAlmMsg(true, 0)
	testBlockDuration(clock, dAlarmSleep, dAlarmSleep)
	comms.chkAlarms <- longButtonAlmMsg(false, time.Second)

	// should have sent a reload message to getAlarms
	e, _ = almStateRead(t, rt.comms.getAlarms)
	assert.Equal(t, e.ID, msgReload)

	testQuit(rt)
}

func TestCheckAlarmsDoubleClick(t *testing.T) {
	rt, clock, comms := testRuntime()
	// events := rt.events.(*testEvents)

	// send pretend like getAlarms ran
	secret := rt.events.generateSecret(rt)
	// send a config error and then a double click into checkAlarms
	comms.chkAlarms <- configErrorMsg(false, secret)

	go runCheckAlarms(rt)
	// wait for a cycle to complete startup loop
	clock.BlockUntil(1)

	// the double click event comes in on another message
	// the duration is longs than the checker, so make sure
	// that we only get 1 effect message
	testBlockDurationCB(clock, dAlarmSleep, time.Second, func(cnt int) {
		comms.chkAlarms <- doubleButtonAlmMsg(true, time.Duration(cnt-1)*dAlarmSleep)
	})

	// should be a bunch of prints
	dE := effectReadAll(comms.effects)

	// also look for the config output
	compares := make([]string, 5)
	prints := []int{
		ePrint,
		ePrintRolling,
		ePrint,
		ePrint,
		ePrintRolling,
	}
	compares[0] = "none"
	compares[1] = "secret"
	compares[2] = "0001"
	compares[3] = "IP:  "
	compares[4] = GetOutboundIP().String()

	for i := range compares {
		assert.Equal(t, prints[i], dE[i].id)
		// log.Printf("%d / %d", len(compares[i]), len(dE[i].val.(displayPrint).s))
		assert.Equal(t, compares[i], dE[i].val.(displayPrint).s)
	}
	assert.Equal(t, len(compares), len(dE))

	testQuit(rt)
}

func TestCheckAlarmsDoubleClickPendingThenCacnel(t *testing.T) {
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
	comms.chkAlarms <- doubleButtonAlmMsg(true, 0)
	testBlockDuration(clock, dAlarmSleep, dAlarmSleep)

	// should have sent cancel prompt
	de := effectReadAll(comms.effects)
	assert.Equal(t, len(de), 2)
	assert.Equal(t, de[0].id, ePrintRolling)
	assert.Equal(t, de[0].val.(displayPrint).s, sCancel)
	assert.Equal(t, de[1].id, ePrint)
	assert.Equal(t, de[1].val.(displayPrint).s, sYorN)
	assert.Assert(t, de[1].val.(displayPrint).cancel != nil)

	testBlockDuration(clock, dAlarmSleep, time.Second)
	// make sure it still says Y : n
	de = effectReadAll(comms.effects)
	assert.Equal(t, len(de), 0)

	// now cancel the pending alarm
	comms.chkAlarms <- mainButtonAlmMsg(true, 0)
	testBlockDuration(clock, dAlarmSleep, time.Second)

	// should have sent cancel prompt
	de = effectReadAll(comms.effects)
	assert.Equal(t, de[0].id, ePrintRolling)
	assert.Equal(t, de[0].val.(displayPrint).s, "-- cancelled --")

	testQuit(rt)
}

func readTrue(c chan bool) bool {
	// should have gotten a cancel print msg on the channel
	cancelled := false
	select {
	case cancelled = <-c:
		// we got it
	default:
	}
	return cancelled
}

func readFalse(c chan bool) bool {
	return !readTrue(c)
}

func TestCheckAlarmsDoubleClickPendingNoCancel(t *testing.T) {
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
	comms.chkAlarms <- doubleButtonAlmMsg(true, 0)
	testBlockDuration(clock, dAlarmSleep, dAlarmSleep)

	// should have sent cancel prompt
	de := effectReadAll(comms.effects)

	assert.Equal(t, de[0].id, ePrintRolling)
	assert.Equal(t, de[0].val.(displayPrint).s, sCancel)
	assert.Equal(t, de[1].id, ePrint)
	assert.Equal(t, de[1].val.(displayPrint).s, sYorN)
	cancelChan := de[1].val.(displayPrint).cancel
	assert.Assert(t, cancelChan != nil)

	// now wait for 10/11 of the cancel and spaces and the Y or n timeout
	testBlockDuration(clock, dAlarmSleep, dCancelTimeout+10*dRollingPrint)
	// should be just short of cancel
	assert.Assert(t, readFalse(cancelChan))
	// one more rolling print cycle
	testBlockDuration(clock, dAlarmSleep, dRollingPrint)
	// checkAlarms close the print channel, the display
	// code counts true and closed as the same
	assert.Assert(t, readTrue(cancelChan))

	// should get next alarm report effects
	es := effectReadAll(rt.comms.effects)
	assert.Equal(t, len(es), 2) // next Al... 8:59
	assert.Equal(t, es[0].id, ePrintRolling)
	assert.Equal(t, es[0].val.(displayPrint).s, sNextALIn)
	assert.Equal(t, es[1].val.(displayPrint).s, " 8:59")

	testQuit(rt)
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
	comms.chkAlarms <- longButtonAlmMsg(true, 0)
	testBlockDuration(clock, dAlarmSleep, 3*dAlarmSleep)
	comms.chkAlarms <- longButtonAlmMsg(false, time.Second)

	// should have sent a reload message to getAlarms
	e, _ := almStateRead(t, rt.comms.getAlarms)
	assert.Equal(t, e.ID, msgReload)

	// pretend like we reloaded new alarms
	alarms, _ = getAlarmsFromService(rt)
	comms.chkAlarms <- alarmsLoadedMsg(1, alarms, true)

	// clear all effects
	effectReadAll(comms.effects)

	// expect a report on the next alarm since we explicitly reloaded
	testBlockDuration(clock, dAlarmSleep, 5*time.Second)

	// read from the effects
	es := effectReadAll(rt.comms.effects)
	assert.Equal(t, len(es), 2) // next Al... 5:59
	assert.Equal(t, es[0].id, ePrintRolling)
	assert.Equal(t, es[0].val.(displayPrint).s, sNextALIn)
	assert.Equal(t, es[1].val.(displayPrint).s, " 5:59")

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
	es := effectReadAll(rt.comms.effects)
	assert.Equal(t, len(es), 2) // next Al... 0:00
	assert.Equal(t, es[0].id, ePrintRolling)
	assert.Equal(t, es[0].val.(displayPrint).s, sNextALIn)
	assert.Equal(t, es[1].val.(displayPrint).s, " 0:00")

	// should not have started countdown
	eA := effectReadAll(rt.comms.effects)
	assert.Equal(t, len(eA), 0)

	// advance until after the alarm starts
	testBlockDurationCB(clock, dAlarmSleep, 2*time.Minute, func(int) {
		le, _ = ledRead(t, rt.comms.leds)
	})

	// should see a play signal and no prints
	e, _ := effectRead(t, rt.comms.effects)
	assert.Equal(t, e.id, eAlarmOn)
	es = effectReadAll(rt.comms.effects)
	assert.Equal(t, len(es), 0)

	// cancel with a button press (and release)
	comms.chkAlarms <- mainButtonAlmMsg(true, 0)
	comms.chkAlarms <- doubleButtonAlmMsg(true, time.Second)
	comms.chkAlarms <- doubleButtonAlmMsg(true, 2*time.Second)
	testBlockDuration(clock, dAlarmSleep, 3*dAlarmSleep)
	comms.chkAlarms <- mainButtonAlmMsg(false, 0)
	testBlockDuration(clock, dAlarmSleep, 2*dAlarmSleep)

	// should see the cancel effect and the next alarm
	es = effectReadAll(rt.comms.effects)
	assert.Equal(t, len(es), 3) // cancel + next al in ... + time

	// wait another minute and make sure nothing else happens
	testBlockDurationCB(clock, dAlarmSleep, time.Minute+time.Second, func(int) {
		le, _ = ledRead(t, rt.comms.leds)
	})

	eA = effectReadAll(rt.comms.effects)
	assert.Equal(t, len(eA), 0)

	// TODO:advance to the next alarm and retest

	// done
	testQuit(rt)
}

func TestCheckAlarmsConfigError(t *testing.T) {
	rt, clock, comms := testRuntime()

	go runCheckAlarms(rt)

	secret := rt.events.generateSecret(rt)
	// send a config error and then a double click into checkAlarms
	comms.chkAlarms <- configErrorMsg(true, secret)
	comms.chkAlarms <- doubleButtonAlmMsg(true, 0)
	comms.chkAlarms <- doubleButtonAlmMsg(true, time.Second)
	comms.chkAlarms <- doubleButtonAlmMsg(true, 2*time.Second)
	comms.chkAlarms <- doubleButtonAlmMsg(false, 0)
	// wait to process
	testBlockDuration(clock, dAlarmSleep, 5*dAlarmSleep)

	// should be a bunch of prints
	dE := effectReadAll(comms.effects)
	// check for secret and IP address
	compares := make([]string, 4)
	prints := []int{
		ePrintRolling,
		ePrint,
		ePrint,
		ePrintRolling,
	}
	compares[0] = "secret"
	compares[1] = secret
	compares[2] = "IP:  "
	compares[3] = GetOutboundIP().String()

	for i := range compares {
		assert.Equal(t, prints[i], dE[i].id)
		assert.Equal(t, compares[i], dE[i].val.(displayPrint).s)
	}

	testQuit(rt)
}

// alarm sort helper
type alarmList []alarm

func (a alarmList) Len() int           { return len(a) }
func (a alarmList) Less(i, j int) bool { return a[i].When.Sub(a[j].When) < 0 }
func (a alarmList) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

func TestMergeAlarms(t *testing.T) {
	rt, _, _ := testRuntime()

	// just test merge alarms
	alarms, _ := getAlarmsFromService(rt)
	alarms2, _ := getAlarmsFromService(rt)

	// test scenarios
	// same list
	results := mergeAlarms(alarms, alarms2)
	assert.Equal(t, len(results), len(alarms))
	assert.Equal(t, len(results), len(alarms2))
	for i := range results {
		assert.Equal(t, results[i], alarms2[i])
		assert.Equal(t, results[i].started, false)
	}
	// some marked as handled
	alarms[2].countdown = true
	alarms[2].started = true
	results = mergeAlarms(alarms, alarms2)
	assert.Equal(t, len(results), len(alarms))
	assert.Equal(t, len(results), len(alarms2))
	assert.Equal(t, results[2].started, true)

	// alarm changed time, marked as handled
	// make sure to re-sort alarms2
	alarms2[2].When = alarms2[2].When.Add(24 * time.Hour)
	modID := alarms2[2].ID
	alarms[2].countdown = true
	alarms[2].started = true
	sort.Sort(alarmList(alarms2))
	results = mergeAlarms(alarms, alarms2)
	lastIndex := len(results) - 1
	assert.Equal(t, len(results), len(alarms))
	assert.Equal(t, len(results), len(alarms2))
	assert.Equal(t, results[lastIndex].ID, modID)
	assert.Equal(t, results[lastIndex].started, false)
	assert.Equal(t, results[lastIndex].countdown, false)

	// one new alarm, in the middle somewhere
	alarms2, _ = getAlarmsFromService(rt)
	almNew := alarm{
		ID:     "test pizza",
		Name:   "test pizza alarm",
		When:   alarms2[2].When.Add(-1 * time.Minute),
		Effect: 0,
		Extra:  "tacos on the side",
	}
	alarms2 = append(alarms2, almNew)
	alarms2[2].countdown = true
	alarms2[2].started = true
	// sort
	sort.Sort(alarmList(alarms2))
	results = mergeAlarms(alarms, alarms2)
	assert.Equal(t, len(results), len(alarms2))
	assert.Equal(t, results[3].countdown, true)
	assert.Equal(t, results[3].started, true)
}

func TestClickWithNoAlarms(t *testing.T) {
	rt, clock, comms := testRuntime()

	go runCheckAlarms(rt)

	// send in a single click
	comms.chkAlarms <- mainButtonAlmMsg(true, 0)
	testBlockDuration(clock, dAlarmSleep, dAlarmSleep)
	comms.chkAlarms <- mainButtonAlmMsg(false, 0)
	testBlockDuration(clock, dAlarmSleep, 2*dAlarmSleep)

	// advance a few seconds, make sure to read the LED channel
	testBlockDurationCB(clock, dAlarmSleep, 10*time.Second, func(cnt int) {
		ledRead(t, rt.comms.leds)
	})

	// should not have anything in effects
	es := effectReadAll(comms.effects)
	assert.Equal(t, len(es), 0)
}

func TestCheckAlarmsFarFuture(t *testing.T) {
	rt, clock, comms := testRuntime()
	// set the clock for > 7d ago
	clock = clockwork.NewFakeClockAt(time.Date(2020, 01, 01, 0, 0, 0, 0, time.UTC))
	rt.clock = clock
	// events := rt.events.(*testEvents)

	go runCheckAlarms(rt)

	// pretend we loaded some alarms
	alarms, _ := getAlarmsFromService(rt)
	comms.chkAlarms <- alarmsLoadedMsg(1, alarms, true)

	// wait for a cycle
	testBlockDuration(clock, dAlarmSleep, dAlarmSleep)

	// should have gotten a bunch of prints in the far future
	es := effectReadAll(comms.effects)
	assert.Equal(t, len(es), 4)
	assert.Equal(t, es[0].id, ePrintRolling)
	assert.Equal(t, es[0].val.(displayPrint).s, sNextAL)
	assert.Equal(t, es[1].id, ePrintRolling)
	assert.Equal(t, es[1].val.(displayPrint).s, "01.26 2020")
	assert.Equal(t, es[2].id, ePrint)
	assert.Equal(t, es[2].val.(displayPrint).s, sAt)
	assert.Equal(t, es[3].id, ePrint)
	assert.Equal(t, es[3].val.(displayPrint).s, "06:00")
	// done
	testQuit(rt)
}

func TestCheckAlarmsReloadTimed(t *testing.T) {
	rt, clock, comms := testRuntime()

	go runCheckAlarms(rt)

	alarms, _ := getAlarmsFromService(rt)
	comms.chkAlarms <- alarmsLoadedMsg(1, alarms, true)

	// wait for a cycle
	testBlockDuration(clock, dAlarmSleep, dAlarmSleep)
	// should have gotten some info
	es := effectReadAll(comms.effects)
	assert.Assert(t, len(es) != 0)

	// getAlarms runs independently, so just send in some more
	comms.chkAlarms <- alarmsLoadedMsg(1, alarms, false)
	testBlockDuration(clock, dAlarmSleep, dAlarmSleep)
	// should have gotten no info
	es = effectReadAll(comms.effects)
	assert.Equal(t, len(es), 0)
}
