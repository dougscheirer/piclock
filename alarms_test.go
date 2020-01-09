package main

import (
	"testing"

	"gotest.tools/assert"
)

func TestAlarms(t *testing.T) {
	// init settings and runtime
	tc := wallClock{}
	s := LoadSettings("./piclock.test.conf", defaultSettings())
	r := initRuntime(rtc{}, tc)

	wg.Add(1)
	go runCheckAlarm(s, r)
	assert.Equal(t, 1, 1)
	close(r.comms.quit)
	wg.Wait()

	// check some things
}
