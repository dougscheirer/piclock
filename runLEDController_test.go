package main

import (
	"testing"
	"time"

	"gotest.tools/assert"
)

// helper func
func compareTo(step int, duty int) bool {
	if (step % 100) < duty {
		return true
	}
	return false
}

// test the LED display modes, not super complicated
func TestLEDControllerModes(t *testing.T) {
	rt, clock, comms := testRuntime()
	leds := rt.led.(*logLed)

	// just set a different mode on each LED, then advance the clock and check all the states
	go runLEDController(rt)
	// wait for one cycle to start
	testBlockDuration(clock, dLEDSleep, dLEDSleep)

	// set a bunch of leds
	comms.leds <- ledMessage(1, modeOff, time.Minute)
	comms.leds <- ledMessage(2, modeOn, time.Minute)
	comms.leds <- ledMessage(3, modeBlink10, time.Minute)
	comms.leds <- ledMessage(4, modeBlink25, time.Minute)
	comms.leds <- ledMessage(5, modeBlink50, time.Minute)
	comms.leds <- ledMessage(6, modeBlink75, time.Minute)
	comms.leds <- ledMessage(7, modeBlink90, time.Minute)

	// advance in steps of 1/100 second for 2 seconds
	for i := 0; i < 200; i++ {
		// log.Printf("Step %d", i)
		testBlockDuration(clock, dLEDSleep, 10*time.Millisecond)
		// #1 is always off
		assert.Equal(t, leds.leds[1], compareTo(i, 0))
		// #2 is always on
		assert.Equal(t, leds.leds[2], compareTo(i, 100))
		// #3 is on from 0->10
		assert.Equal(t, leds.leds[3], compareTo(i, 90))
		// #4 is on from 0->25
		assert.Equal(t, leds.leds[4], compareTo(i, 75))
		// #5 is on from 0->50
		assert.Equal(t, leds.leds[5], compareTo(i, 50))
		// #6 is on from 0->75
		assert.Equal(t, leds.leds[6], compareTo(i, 25))
		// #7 is on from 0->90
		assert.Equal(t, leds.leds[7], compareTo(i, 10))
	}

	// disable logging
	leds.disableLog = true
	// fast-forward 58 seconds and 1 cycle
	testBlockDuration(clock, dLEDSleep, 58*time.Second+dLEDSleep)
	// all should be off now (and they don't come back on)
	for i := 1; i < 8; i++ {
		assert.Equal(t, leds.leds[i], false)
	}
	testBlockDuration(clock, dLEDSleep, 500*time.Millisecond)
	for i := 1; i < 8; i++ {
		assert.Equal(t, leds.leds[i], false)
	}

	//done
	testQuit(rt)
}

func TestLEDControllerOnForever(t *testing.T) {
	rt, clock, comms := testRuntime()
	leds := rt.led.(*logLed)

	go runLEDController(rt)

	// turn an LED on forever
	comms.leds <- ledOn(2)

	// wait for a while
	testBlockDuration(clock, dLEDSleep, time.Minute)

	// should have 1 audit message
	assert.Equal(t, len(leds.audit), 1)
	assert.Equal(t, leds.leds[2], true)

	//done
	testQuit(rt)
}
