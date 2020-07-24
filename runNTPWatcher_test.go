package main

import (
	"testing"
	"time"

	"gotest.tools/assert"
)

/* things that runNTPWatcher does:

checks clock time against internet availble time (TZ by IP)
displays message when time is off by more than 5m

*/

func TestCheckTimeOK(t *testing.T) {
	rt, clock, comms := testRuntime()

	// set the "IP time" to the same as clock time
	ntp := rt.ntpCheck.(*testNtpChecker)
	ntp.curtime = clock.Now()

	go runNTPWatcher(rt)
	clock.BlockUntil(1)

	// no messages
	es := effectReadAll(comms.effects)
	assert.Equal(t, len(es), 0)

	// done
	testQuit(rt)
}

func TestCheckTimeAhead(t *testing.T) {
	rt, clock, comms := testRuntime()

	// set the "IP time" to the same as clock time
	ntp := rt.ntpCheck.(*testNtpChecker)
	ntp.curtime = clock.Now().Add(time.Hour)

	go runNTPWatcher(rt)
	clock.BlockUntil(1)

	// messages
	es := effectReadAll(comms.effects)
	assert.Equal(t, len(es), 1)
	assert.Equal(t, es[0].val.(displayPrint).s, sNeedSync)

	// done
	testQuit(rt)
}

func TestCheckTimeBehind(t *testing.T) {
	rt, clock, comms := testRuntime()

	// set the "IP time" to the same as clock time
	ntp := rt.ntpCheck.(*testNtpChecker)
	ntp.curtime = clock.Now().Add(-time.Hour)

	go runNTPWatcher(rt)
	clock.BlockUntil(1)

	// messages
	es := effectReadAll(comms.effects)
	assert.Equal(t, len(es), 1)
	assert.Equal(t, es[0].val.(displayPrint).s, sNeedSync)

	// done
	testQuit(rt)
}

func TestCheckTimeOffALittle(t *testing.T) {
	rt, clock, comms := testRuntime()

	// set the "IP time" to the same as clock time
	ntp := rt.ntpCheck.(*testNtpChecker)
	ntp.curtime = clock.Now().Add(-time.Second)

	go runNTPWatcher(rt)
	clock.BlockUntil(1)

	// no messages
	es := effectReadAll(comms.effects)
	assert.Equal(t, len(es), 0)
	// assert.Equal(t, es[0].val.(displayPrint).s, sNeedSync)

	// done
	testQuit(rt)
}

func TestCheckTimeOffThenOK(t *testing.T) {
	rt, clock, comms := testRuntime()

	// set the "IP time" to the same as clock time
	ntp := rt.ntpCheck.(*testNtpChecker)
	ntp.curtime = clock.Now().Add(6 * time.Minute)

	go runNTPWatcher(rt)
	clock.BlockUntil(1)

	// messages
	es := effectReadAll(comms.effects)
	assert.Equal(t, len(es), 1)
	assert.Equal(t, es[0].val.(displayPrint).s, sNeedSync)

	// block for sleep time and adjust time
	ntp.curtime = clock.Now().Add(dNTPCheckSleep + 6*time.Minute)
	clock.BlockUntil(1)
	clock.BlockUntil(1)

	es = effectReadAll(comms.effects)
	assert.Equal(t, len(es), 0)

	// done
	testQuit(rt)
}
