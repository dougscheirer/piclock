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
		for _, v := range leds {
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
				continue
			}

			if v.duration == 0 {
				// 0 duration means "forever"
				continue
			}

			// duration expired means turn it off
			if runtime.rtc.now().Sub(v.lastUpdate) > v.duration {
				if v.curMode != modeOff {
					setLED(v.pin, false)
					// negative duration is expired
					v.duration = -1
					v.curMode = modeOff
					v.lastUpdate = runtime.rtc.now()
					continue
				}
			}

			timeInState := runtime.rtc.now().Sub(v.lastUpdate)
			var upTime, downTime time.Duration

			switch v.mode {
			case modeBlink50:
				upTime = 50
				downTime = 50
			case modeBlink75:
				upTime = 25
				downTime = 75
			case modeBlink90:
				upTime = 10
				downTime = 90
			case modeOn:
			case modeOff:
			default:
				// nothing to do
				continue
			}

			if v.curMode == modeOff {
				if timeInState > downTime*time.Microsecond {
					setLED(v.pin, true)
					v.curMode = modeOn
					v.lastUpdate = runtime.rtc.now()
				}
			} else {
				if timeInState > upTime*time.Millisecond {
					setLED(v.pin, false)
					v.curMode = modeOff
					v.lastUpdate = runtime.rtc.now()
				}
			}
		}

		// sleep for a bit (1/10s is our lowest resolution)
		time.Sleep(100 * time.Millisecond)
	}
}
