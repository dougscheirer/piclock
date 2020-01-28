package main

import (
	"testing"
	"time"

	"gotest.tools/assert"
)

/* different modes to test via effects channel
case eClock:
case eCountdown:
case eAlarmError:
case ePrint:
case eAlarmOn:
case eAlarmOff:
case eMainButton:
*/

func TestClockMode(t *testing.T) {
	rt, clock, _ := testRuntime()
	ld := rt.display.(*logDisplay)

	clock.Advance(9*time.Hour + 15*time.Minute)
	go runEffects(rt)

	// wait for two cycles (the display loop is a little weird)
	testBlockDuration(clock, dEffectSleep, 2*dEffectSleep)

	// check the logDisplay
	assert.Equal(t, ld.curDisplay, " 9:15")

	// advance by a few minutes
	testBlockDuration(clock, dEffectSleep, 4*time.Minute)

	assert.Equal(t, ld.curDisplay, " 9:19")

	// done
	testQuit(rt)
}

func TestClockModeButton(t *testing.T) {
	rt, clock, _ := testRuntime()
	ld := rt.display.(*logDisplay)

	clock.Advance(9*time.Hour + 15*time.Minute)
	go runEffects(rt)

	// wait for two cycles (the display loop is a little weird)
	testBlockDuration(clock, dEffectSleep, 2*dEffectSleep)

	// check the logDisplay
	assert.Equal(t, ld.curDisplay, " 9:15")

	// press the button
	rt.comms.effects <- mainButtonEffect(true, 1)
	testBlockDuration(clock, dEffectSleep, 1)
	// should be time with a dot
	assert.Equal(t, ld.curDisplay, " 9:15.")

	// un-press
	rt.comms.effects <- mainButtonEffect(false, 1)
	testBlockDuration(clock, dEffectSleep, 1)
	// should be time with a dot
	assert.Equal(t, ld.curDisplay, " 9:15")

	// done
	testQuit(rt)
}

func TestClockModeCountdown(t *testing.T) {
	rt, clock, _ := testRuntime()
	ld := rt.display.(*logDisplay)

	clock.Advance(9*time.Hour + 15*time.Minute)
	go runEffects(rt)

	// send countdown message
	alm := alarm{
		ID:        "xoxoxo",
		Name:      "test alarm",
		When:      clock.Now().Add(time.Minute),
		Effect:    almMusic,
		Extra:     "pizza",
		started:   false,
		countdown: true,
	}

	rt.comms.effects <- setCountdownMode(alm)

	// advance for just under 1 second at the countdown rate
	testBlockDuration(clock, dEffectSleep, 900*time.Millisecond)

	// we did 10 and started at 59.9
	assert.Equal(t, ld.curDisplay, "59.1")
	assert.Equal(t, len(ld.audit), 9)

	// now cancel
	alm.started = true
	rt.comms.effects <- cancelAlarmMode()

	// advance and back to the clock
	testBlockDuration(clock, dEffectSleep, dEffectSleep)
	assert.Equal(t, ld.curDisplay, " 9:15")

	// make sure the alarm effect did not fire
	s := rt.sounds.(*noSounds)
	assert.Equal(t, s.playMP3Cnt, 0)
	assert.Equal(t, s.playItCnt, 0)

	// done
	testQuit(rt)
}

func TestClockModeAlarmError(t *testing.T) {
	rt, clock, _ := testRuntime()
	ld := rt.display.(*logDisplay)

	clock.Advance(9*time.Hour + 15*time.Minute)
	go runEffects(rt)

	rt.comms.effects <- alarmError(3 * time.Second)

	// advance to see the msg
	testBlockDuration(clock, dEffectSleep, dEffectSleep)
	assert.Equal(t, ld.curDisplay, "Err")

	// now wait
	testBlockDuration(clock, dEffectSleep, 3*time.Second)
	assert.Equal(t, ld.curDisplay, " 9:15")

	// done
	testQuit(rt)
}

func TestClockModePrint(t *testing.T) {
	rt, clock, _ := testRuntime()
	ld := rt.display.(*logDisplay)

	clock.Advance(9*time.Hour + 15*time.Minute)
	go runEffects(rt)

	rt.comms.effects <- printEffect("bob", time.Second)

	// advance to see the msg
	testBlockDuration(clock, dEffectSleep, dEffectSleep)
	assert.Equal(t, ld.curDisplay, "bob")

	// now wait
	testBlockDuration(clock, dEffectSleep, time.Second+dEffectSleep)
	assert.Equal(t, ld.curDisplay, " 9:15")

	// done
	testQuit(rt)
}

func TestClockModeAlarmOn(t *testing.T) {
	rt, clock, _ := testRuntime()
	ld := rt.display.(*logDisplay)

	clock.Advance(9*time.Hour + 15*time.Minute)
	go runEffects(rt)

	// advance to see the time
	testBlockDuration(clock, dEffectSleep, 2*dEffectSleep)
	assert.Equal(t, ld.curDisplay, " 9:15")

	// send alarm message
	alm := alarm{
		ID:        "xoxoxo",
		Name:      "test alarm",
		When:      clock.Now(),
		Effect:    almMusic,
		Extra:     "pizza",
		started:   false,
		countdown: true,
	}

	rt.comms.effects <- setAlarmMode(alm)

	// now wait
	testBlockDuration(clock, dEffectSleep, dEffectSleep)
	assert.Equal(t, ld.curDisplay, almDisplay1)
	// wait for the other one
	testBlockDuration(clock, dEffectSleep, 3*time.Second)
	assert.Equal(t, ld.curDisplay, almDisplay2)

	// make sure the alarm effect did fire
	s := rt.sounds.(*noSounds)
	assert.Equal(t, s.playMP3Cnt, 1)
	assert.Equal(t, s.playItCnt, 0)

	// cancel alarm
	rt.comms.effects <- cancelAlarmMode()

	// now wait
	testBlockDuration(clock, dEffectSleep, dEffectSleep)
	assert.Equal(t, ld.curDisplay, " 9:15")

	// done
	testQuit(rt)
}

func TestClockModeAlarmOver(t *testing.T) {
	rt, clock, _ := testRuntime()
	ld := rt.display.(*logDisplay)

	clock.Advance(9*time.Hour + 15*time.Minute)
	go runEffects(rt)

	// advance to see the time
	testBlockDuration(clock, dEffectSleep, 2*dEffectSleep)
	assert.Equal(t, ld.curDisplay, " 9:15")

	// send alarm message
	alm := alarm{
		ID:        "xoxoxo",
		Name:      "test alarm",
		When:      clock.Now(),
		Effect:    almMusic,
		Extra:     "pizza",
		started:   false,
		countdown: true,
	}

	rt.comms.effects <- setAlarmMode(alm)

	// now wait
	testBlockDuration(clock, dEffectSleep, dEffectSleep)
	assert.Equal(t, ld.curDisplay, "_-_-")
	// make sure the alarm effect did fire
	s := rt.sounds.(*noSounds)
	assert.Equal(t, s.playMP3Cnt, 1)
	assert.Equal(t, s.playItCnt, 0)

	// signal the play completed
	ns := rt.sounds.(*noSounds)
	ns.done <- true

	// now wait
	testBlockDuration(clock, dEffectSleep, dEffectSleep)
	assert.Equal(t, ld.curDisplay, " 9:15")

	// now play tones
	alm.Effect = almTones

	rt.comms.effects <- setAlarmMode(alm)

	// now wait
	testBlockDuration(clock, dEffectSleep, dEffectSleep)
	assert.Equal(t, ld.curDisplay, "_-_-")
	// make sure the alarm effect did fire
	s = rt.sounds.(*noSounds)
	assert.Equal(t, s.playMP3Cnt, 1)
	assert.Equal(t, s.playItCnt, 1)

	// signal the play completed
	ns = rt.sounds.(*noSounds)
	ns.done <- true

	// now wait
	testBlockDuration(clock, dEffectSleep, dEffectSleep)
	assert.Equal(t, ld.curDisplay, " 9:15")

	// done
	testQuit(rt)
}

func TestPrintDoesNotOverrideAlarm(t *testing.T) {
	rt, clock, _ := testRuntime()
	ld := rt.display.(*logDisplay)

	clock.Advance(9*time.Hour + 15*time.Minute)
	go runEffects(rt)

	// advance to see the time
	testBlockDuration(clock, dEffectSleep, 2*dEffectSleep)
	assert.Equal(t, ld.curDisplay, " 9:15")

	// send alarm message
	alm := alarm{
		ID:        "xoxoxo",
		Name:      "test alarm",
		When:      clock.Now(),
		Effect:    almMusic,
		Extra:     "pizza",
		started:   false,
		countdown: true,
	}

	rt.comms.effects <- setAlarmMode(alm)

	// now wait
	testBlockDuration(clock, dEffectSleep, dEffectSleep)
	assert.Equal(t, ld.curDisplay, "_-_-")
	// make sure the alarm effect did fire
	s := rt.sounds.(*noSounds)
	assert.Equal(t, s.playMP3Cnt, 1)
	assert.Equal(t, s.playItCnt, 0)

	// now tell it to print stuff (this is what checkAlarms will do)
	// ignore the print command while the alarm is firing, but print
	// it later when the alarm is done
	rt.comms.effects <- printEffect("pie", 2*time.Second)
	testBlockDuration(clock, dEffectSleep, 20*time.Second)

	assert.Equal(t, ld.curDisplay, "_-_-")
	// now signal the alarm finished
	ns := rt.sounds.(*noSounds)
	ns.done <- true

	// now wait
	testBlockDuration(clock, dEffectSleep, dEffectSleep)
	assert.Equal(t, ld.curDisplay, "pie")

	// wait for it to clear to the time
	testBlockDuration(clock, dEffectSleep, 2*time.Second+dEffectSleep)
	assert.Equal(t, ld.curDisplay, " 9:15")
}
