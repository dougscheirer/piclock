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
case eAlarm:
case eMainButton:
*/

func TestClockMode(t *testing.T) {
	rt, clock, _ := testRuntime()
	ld := rt.display.(*logDisplay)

	clock.Advance(9*time.Hour + 15*time.Minute)
	go runEffects(rt)

	// wait for two cycles (the display loop is a little weird)
	ds := rt.settings.GetDuration(sSleep)
	testBlockDuration(clock, ds, 2*ds)

	// check the logDisplay
	assert.Assert(t, ld.curDisplay == " 9:15")

	// advance by a few minutes
	testBlockDuration(clock, ds, 4*time.Minute)

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
	ds := rt.settings.GetDuration(sSleep)
	testBlockDuration(clock, ds, 2*ds)

	// check the logDisplay
	assert.Assert(t, ld.curDisplay == " 9:15")

	// advance by a few minutes
	testBlockDuration(clock, ds, 4*time.Minute)

	assert.Assert(t, ld.curDisplay == " 9:19")

	// done
	close(rt.comms.quit)
}
