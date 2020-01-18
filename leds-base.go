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
<<<<<<< HEAD
	startTime  time.Time // runtime setting, when we initiated
=======
>>>>>>> d9838022f2481ff2dc55479298935ecea483d41a
}

func init() {
	// wait group for runLEDController
	wg.Add(1)
}

func ledMessage(pin int, mode int, duration time.Duration) ledEffect {
<<<<<<< HEAD
	return ledEffect{pin: pin, mode: mode, duration: duration, startTime: time.Time{}}
=======
	return ledEffect{pin: pin, mode: mode, duration: duration}
>>>>>>> d9838022f2481ff2dc55479298935ecea483d41a
}

func diffLEDEffect(effect1 ledEffect, effect2 ledEffect) bool {
	return effect1.mode != effect2.mode || effect1.duration != effect2.duration ||
<<<<<<< HEAD
		effect1.pin != effect2.pin || effect1.startTime != effect2.startTime
=======
		effect1.pin != effect2.pin
>>>>>>> d9838022f2481ff2dc55479298935ecea483d41a
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
			} else {
				// it's new, add to the leds map
				leds[msg.pin] = setLEDEffect(msg)
			}
		default:
		}
		// for anything that we're doing blink on, see if it's time to toggle
		// also anything that is modeUnset needs to be initiated
		for i, v := range leds {
			// negative duration is "ignore"
			if v.duration < 0 {
				continue
			}

			if v.curMode == modeUnset {
				// transform broader categories of mode to on/off
				if v.mode == modeOff {
					setLED(v.pin, false)
					v.curMode = modeOff
				} else {
					setLED(v.pin, true)
					v.curMode = modeOn
				}
				v.lastUpdate = runtime.rtc.now()
				v.startTime = v.lastUpdate
				// if it's just "off" or "on" set the duration to -1 so we never re-check
				if v.mode == modeOff || v.mode == modeOn {
					v.duration = -1
				}
				leds[i] = v
				continue
			}

			// duration expired means turn it off
			if v.duration > 0 && runtime.rtc.now().Sub(v.startTime) > v.duration {
				if v.curMode != modeOff {
					setLED(v.pin, false)
					// negative duration is expired
					// TODO: remove from the map to make processing faster
					v.duration = -1
					v.curMode = modeOff
					v.lastUpdate = time.Time{}
					v.startTime = time.Time{}
					leds[i] = v
					continue
				}
			}

			timeInState := runtime.rtc.now().Sub(v.lastUpdate)
			var upTime, downTime time.Duration

			switch v.mode {
			case modeBlink50:
				upTime = 500
				downTime = 500
			case modeBlink75:
				upTime = 250
				downTime = 750
			case modeBlink90:
				upTime = 100
				downTime = 900
			case modeOn:
			case modeOff:
			default:
				// nothing to do
				continue
			}

			if v.curMode == modeOff {
				if timeInState > downTime*time.Millisecond {
					setLED(v.pin, true)
					v.curMode = modeOn
					v.lastUpdate = runtime.rtc.now()
					leds[i] = v
				}
			} else {
				if timeInState > upTime*time.Millisecond {
					setLED(v.pin, false)
					v.curMode = modeOff
					v.lastUpdate = runtime.rtc.now()
					leds[i] = v
				}
			}
		}

		// sleep for a bit (1/10s is our lowest resolution)
		time.Sleep(100 * time.Millisecond)
	}
}
