package main

import (
	"log"
	"time"
)

const (
	modeOff = iota
	modeOn
	modeBlink50 // 50% cycle/sec
	modeBlink75 // 75% off/sec
	modeBlink90 // 90% off/sec
	modeUnset   // undetermined state
)

type ledEffect struct {
	pin        int
	mode       int
	duration   time.Duration
	curMode    int       // runtime setting, on or off
	lastUpdate time.Time // runtime setting, last time we changed the state
}

func init() {
	// wait group for runLEDController
	wg.Add(1)
}

func ledMessage(pin int, mode int, duration time.Duration) ledEffect {
	return ledEffect{pin: pin, mode: mode, duration: duration}
}

func diffLEDEffect(effect1 ledEffect, effect2 ledEffect) bool {
	return effect1.mode != effect2.mode || effect1.duration != effect2.duration ||
		effect1.pin != effect2.pin
}

func setLEDEffect(effect ledEffect) ledEffect {
	// clear the runtime info
	effect.curMode = modeUnset
	effect.lastUpdate = time.Time{}
	return effect
}

func runLEDController(settings *settings, runtime runtimeConfig) {
	defer wg.Done()
	defer func() {
		log.Printf("Exitings runLEDController")
	}()

	leds := make(map[int]ledEffect)

	comms := runtime.comms

	for true {
		select {
		case <-comms.quit:
			log.Printf("Got a quit signal in runLEDController")
			return
		case msg := <-comms.leds:
			// TODO: find in leds, determine if we need to change the state
			if val, ok := leds[msg.pin]; ok {
				// if the state is changed, set the new effect state
				if diffLEDEffect(val, msg) {
					leds[msg.pin] = setLEDEffect(msg)
				}
			}
		default:
			continue
		}
		// for anything that we're doing blink on, see if it's time to toggle
		// also anything that is modeUnset needs to be initiated
		for k, v := range leds {
			if v.curMode == modeUnset {
				// transform broader categories of mode to on/off
				mode := modeOn
				if v.mode == modeOff {
					mode = modeOff
				}
				setLED(v.pin, v.mode)
				v.curMode = mode
				v.lastUpdate = time.Now()
			} else if time.Now()-v.lastUpdate > v.duration {
				// duration timeout, turn it off
				if v.curMode != modeOff {
					setLED(v.pin, modeOff)
					// negative duration is expired
					v.duration = -1
					v.curMode := modeOff
					v.lastUpdate = time.Now()
				}
			}
		}

		// sleep for a bit (1/10s is our lowest resolution)
		time.Sleep(100 * time.Millisecond)
	}
}
