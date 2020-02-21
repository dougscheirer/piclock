package main

import (
	"testing"
	"time"

	"dscheirer.com/piclock/sevenseg_backpack"

	"gotest.tools/assert"
)

/* different modes to test via effects channel
case eClock:
case eCountdown:
case eAlarmError:
case ePrint:
case ePrintRolling:
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
	testBlockDuration(clock, dEffectSleep, dEffectSleep)

	// check the logDisplay
	assert.Equal(t, ld.curDisplay, " 9:15")

	// advance by a few minutes
	testBlockDuration(clock, dEffectSleep, 4*time.Minute)

	assert.Equal(t, ld.curDisplay, " 9:19")
	assert.Equal(t, len(ld.auditErrors), 0)

	// done
	testQuit(rt)
}

func TestClockModeButton(t *testing.T) {
	rt, clock, _ := testRuntime()
	ld := rt.display.(*logDisplay)

	clock.Advance(9*time.Hour + 15*time.Minute)
	go runEffects(rt)

	testBlockDuration(clock, dEffectSleep, dEffectSleep)

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
	assert.Equal(t, len(ld.auditErrors), 0)

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
	assert.Equal(t, len(ld.auditErrors), 0)

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
	assert.Equal(t, len(ld.auditErrors), 0)

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
	assert.Equal(t, len(ld.auditErrors), 0)

	// done
	testQuit(rt)
}

func TestClockModeAlarmOn(t *testing.T) {
	rt, clock, _ := testRuntime()
	ld := rt.display.(*logDisplay)

	clock.Advance(9*time.Hour + 15*time.Minute)
	go runEffects(rt)

	// advance to see the time
	testBlockDuration(clock, dEffectSleep, dEffectSleep)
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
	assert.Equal(t, len(ld.auditErrors), 0)

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
	assert.Equal(t, len(ld.auditErrors), 0)

	// done
	testQuit(rt)
}

func TestPrintDoesNotOverrideAlarm(t *testing.T) {
	rt, clock, _ := testRuntime()
	ld := rt.display.(*logDisplay)

	clock.Advance(9*time.Hour + 15*time.Minute)
	go runEffects(rt)

	// advance to see the time
	testBlockDuration(clock, dEffectSleep, dEffectSleep)
	assert.Equal(t, ld.curDisplay, " 9:15")

	// send alarm message
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

	// should start the countdown
	testBlockDuration(clock, dEffectSleep, dEffectSleep)
	assert.Equal(t, ld.curDisplay, "59.9")

	// advance to the last 10 seconds
	clock.Advance(50 * time.Second)
	testBlockDuration(clock, dEffectSleep, dEffectSleep)
	assert.Equal(t, ld.curDisplay, "9.9")
	assert.Equal(t, ld.blinkRate, uint8(sevenseg_backpack.BLINK_2HZ))

	// advance to the alarm time, send the alarm
	clock.Advance(50 * time.Second)
	rt.comms.effects <- setAlarmMode(alm)
	testBlockDuration(clock, dEffectSleep, dEffectSleep)
	assert.Equal(t, ld.curDisplay, "_-_-")
	assert.Equal(t, ld.blinkRate, uint8(sevenseg_backpack.BLINK_OFF))

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
	assert.Equal(t, ld.curDisplay, " 9:17")
	assert.Equal(t, len(ld.auditErrors), 0)

	testQuit(rt)
}

func TestClockModeDoubleClick(t *testing.T) {
	rt, clock, _ := testRuntime()
	ld := rt.display.(*logDisplay)

	go runEffects(rt)

	testBlockDuration(clock, dEffectSleep, dEffectSleep)

	// check the logDisplay
	assert.Equal(t, ld.curDisplay, " 0:00")

	// press the button
	rt.comms.effects <- doubleButtonEffect(true, 0)
	testBlockDuration(clock, dEffectSleep, 1)

	// should be time with no dot
	assert.Equal(t, ld.curDisplay, " 0:00")

	// un-press
	rt.comms.effects <- doubleButtonEffect(false, time.Second)
	testBlockDuration(clock, dEffectSleep, dEffectSleep)
	// should be time with a dot
	assert.Equal(t, ld.curDisplay, " 0:00")
	assert.Equal(t, len(ld.auditErrors), 0)

	// done
	testQuit(rt)
}

func TestBadPrint(t *testing.T) {
	rt, clock, comms := testRuntime()
	ld := rt.display.(*logDisplay)

	// print something with a character that is impossible
	comms.effects <- printEffect("pizz", 0)

	go runEffects(rt)

	testBlockDuration(clock, dEffectSleep, dEffectSleep)

	assert.Equal(t, len(ld.auditErrors), 1)
}

func TestPrintRolling(t *testing.T) {
	rt, clock, comms := testRuntime()
	ld := rt.display.(*logDisplay)

	longString := "hello....hi....nope"
	comms.effects <- printRollingEffect(longString, 100*time.Millisecond) // the duration is the time between each roll

	go runEffects(rt)

	testBlockDuration(clock, 100*time.Millisecond, 100*time.Millisecond*time.Duration(len(longString)+5)) // len(string) + blanks on both ends

	assert.Equal(t, len(ld.audit), len(longString)+5)
	assert.Equal(t, ld.audit[0], "    ")
	assert.Equal(t, ld.audit[5], "ello.")
	assert.Equal(t, ld.audit[9], "....")

	assert.Equal(t, len(ld.auditErrors), 0)
}

func TestPrintRolling2(t *testing.T) {
	rt, clock, comms := testRuntime()
	ld := rt.display.(*logDisplay)

	longString := "build"
	comms.effects <- printRollingEffect(longString, 100*time.Millisecond) // the duration is the time between each roll

	go runEffects(rt)

	testBlockDuration(clock, 100*time.Millisecond, 100*time.Millisecond*time.Duration(len(longString)+5)) // len(string) + blanks on both ends

	assert.Equal(t, len(ld.auditErrors), 0)

	assert.Equal(t, len(ld.audit), len(longString)+5)
	assert.Equal(t, ld.audit[0], "    ")
	assert.Equal(t, ld.audit[5], "uild")
}

func TestPrintWithCancel(t *testing.T) {
	rt, clock, comms := testRuntime()
	ld := rt.display.(*logDisplay)

	print := "99:99"
	cancel := make(chan bool, 1)
	comms.effects <- printCancelableEffect(print, 5*time.Second, cancel)

	go runEffects(rt)

	// sleep for 1 second, cancel, sleep for a dEffectSleep and check the display
	testBlockDuration(clock, dEffectSleep, time.Second)
	assert.Equal(t, len(ld.audit), 1)
	assert.Equal(t, ld.audit[0], "99:99")
	cancel <- true
	testBlockDuration(clock, dEffectSleep, 2*dEffectSleep) // sleep for 2, one for the cancel handler, one for the next clock refresh

	testBlockDuration(clock, dEffectSleep, time.Second)
	assert.Equal(t, len(ld.audit), 2)
	assert.Equal(t, ld.audit[1], " 0:00")

	assert.Equal(t, len(ld.auditErrors), 0)
}

func TestPrintRollingWithCancel(t *testing.T) {
	rt, clock, comms := testRuntime()
	ld := rt.display.(*logDisplay)

	print := "9999"
	cancel := make(chan bool, 1)
	comms.effects <- printCancelableRollingEffect(print, dRollingPrint, cancel)

	go runEffects(rt)

	// sleep for 1 second, cancel, sleep for a 3 rolling print cycles and check the display
	testBlockDuration(clock, dEffectSleep, 3*dRollingPrint)
	cancel <- true
	testBlockDuration(clock, dEffectSleep, 2*dEffectSleep) // sleep for 2, one for the cancel handler, one for the next clock refresh

	// audit log should be 3 prints then the current time (0:00)
	assert.Equal(t, len(ld.audit), 5)
	assert.Equal(t, ld.audit[0], "    ")
	assert.Equal(t, ld.audit[1], "   9")
	assert.Equal(t, ld.audit[2], "  99")
	assert.Equal(t, ld.audit[3], " 999")
	assert.Equal(t, ld.audit[4], " 0:00")

	assert.Equal(t, len(ld.auditErrors), 0)
}
