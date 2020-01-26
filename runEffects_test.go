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
	assert.Assert(t, ld.curDisplay == " 9:15")

	// advance by a few minutes
	testBlockDuration(clock, dEffectSleep, 4*time.Minute)

	assert.Assert(t, ld.curDisplay == " 9:19")

	// done
	close(rt.comms.quit)
}

func TestClockModeButton(t *testing.T) {
	rt, clock, _ := testRuntime()
	ld := rt.display.(*logDisplay)

	clock.Advance(9*time.Hour + 15*time.Minute)
	go runEffects(rt)

	// wait for two cycles (the display loop is a little weird)
	testBlockDuration(clock, dEffectSleep, 2*dEffectSleep)

	// check the logDisplay
	assert.Assert(t, ld.curDisplay == " 9:15")

	// press the button
	rt.comms.effects <- mainButtonEffect(true, 1)
	testBlockDuration(clock, dEffectSleep, 1)
	// should be time with a dot
	assert.Assert(t, ld.curDisplay == " 9:15.")

	// un-press
	rt.comms.effects <- mainButtonEffect(false, 1)
	testBlockDuration(clock, dEffectSleep, 1)
	// should be time with a dot
	assert.Assert(t, ld.curDisplay == " 9:15")

	// done
	close(rt.comms.quit)
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
	assert.Assert(t, ld.curDisplay == "59.0")
	assert.Assert(t, len(ld.audit) == 10)

	// now cancel
	alm.started = true
	rt.comms.effects <- cancelAlarmMode(alm)

	// advance and back to the clock
	testBlockDuration(clock, dEffectSleep, dEffectSleep)
	assert.Assert(t, ld.curDisplay == " 9:15")

	// done
	close(rt.comms.quit)
}

func TestClockModeAlarmError(t *testing.T) {
	rt, clock, _ := testRuntime()
	ld := rt.display.(*logDisplay)

	clock.Advance(9*time.Hour + 15*time.Minute)
	go runEffects(rt)

	rt.comms.effects <- alarmError(3 * time.Second)

	// advance to see the msg
	testBlockDuration(clock, dEffectSleep, dEffectSleep)
	assert.Assert(t, ld.curDisplay == "Err")

	// now wait
	testBlockDuration(clock, dEffectSleep, 3*time.Second)
	assert.Assert(t, ld.curDisplay == " 9:15")

	// done
	close(rt.comms.quit)
}

func TestClockModePrint(t *testing.T) {
	rt, clock, _ := testRuntime()
	ld := rt.display.(*logDisplay)

	clock.Advance(9*time.Hour + 15*time.Minute)
	go runEffects(rt)

	rt.comms.effects <- printEffect("bob", time.Second)

	// advance to see the msg
	testBlockDuration(clock, dEffectSleep, dEffectSleep)
	assert.Assert(t, ld.curDisplay == "bob")

	// now wait
	testBlockDuration(clock, dEffectSleep, time.Second)
	assert.Assert(t, ld.curDisplay == " 9:15")

	// done
	close(rt.comms.quit)
}

func TestClockModeAlarmOn(t *testing.T) {
	rt, clock, _ := testRuntime()
	ld := rt.display.(*logDisplay)

	clock.Advance(9*time.Hour + 15*time.Minute)
	go runEffects(rt)

	// advance to see the time
	testBlockDuration(clock, dEffectSleep, dEffectSleep)
	assert.Assert(t, ld.curDisplay == " 9:15")

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
	assert.Assert(t, ld.curDisplay == "_-_-")

	// cancel alarm
	rt.comms.effects <- cancelAlarmMode(alm)

	// now wait
	testBlockDuration(clock, dEffectSleep, dEffectSleep)
	assert.Assert(t, ld.curDisplay == " 9:15")

	// done
	close(rt.comms.quit)
}
