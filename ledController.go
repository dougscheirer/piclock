package main

import (
	"log"
	"time"
)

const (
	modeOff = iota
	modeOn
	modeBlink10 // 10% off/sec
	modeBlink25 // 25% off/sec
	modeBlink50 // 50% cycle/sec
	modeBlink75 // 75% off/sec
	modeBlink90 // 90% off/sec
	modeUnset   // undetermined state
)

type ledEffect struct {
	pin        int
	mode       int
	duration   time.Duration
	force      bool      // ignore current state, just do it
	curMode    int       // rt setting, on or off
	lastUpdate time.Time // rt setting, last time we changed the state
	startTime  time.Time // rt setting, when we initiated
}

func init() {
	// wait group for runLEDController
	wg.Add(1)
}

func ledMessage(pin int, mode int, duration time.Duration) ledEffect {
	return ledEffect{pin: pin, mode: mode, duration: duration, startTime: time.Time{}, force: false}
}

func ledMessageForce(pin int, mode int, duration time.Duration) ledEffect {
	return ledEffect{pin: pin, mode: mode, duration: duration, startTime: time.Time{}, force: true}
}

func ledOn(pin int) ledEffect {
	return ledMessage(pin, modeOn, 0)
}

func ledOff(pin int) ledEffect {
	return ledMessage(pin, modeOff, 0)
}

func diffLEDEffect(effect1 ledEffect, effect2 ledEffect) bool {
	return effect1.mode != effect2.mode || (effect1.duration != effect2.duration && effect1.duration > 0 && effect2.duration > 0) ||
		effect1.pin != effect2.pin || (effect1.startTime != effect2.startTime && effect1.duration > 0 && effect2.duration > 0)
}

func setLEDEffect(effect ledEffect) ledEffect {
	// clear the rt info
	effect.curMode = modeUnset
	effect.lastUpdate = time.Time{}
	effect.force = false // this is not part of the rt, just an indicator in the message
	return effect
}

func runLEDController(rt runtimeConfig) {
	defer wg.Done()
	defer func() {
		log.Printf("Exiting runLEDController")
	}()

	comms := rt.comms
	leds := make(map[int]ledEffect)

	rt.led.init()

	for true {
		// read all incoming messages at once
		keepReading := true
		for keepReading {
			select {
			case <-comms.quit:
				log.Printf("Got a quit signal in runLEDController")
				return
			case msg := <-comms.leds:
				// find in leds, determine if we need to change the state
				if val, ok := leds[msg.pin]; ok {
					// if the state is changed, set the new effect state
					if val.force || diffLEDEffect(val, msg) {
						log.Printf("Received led message: %v", msg)
						leds[msg.pin] = setLEDEffect(msg)
					} else {
						// log.Println("Duplicate message")
					}
				} else {
					// it's new, add to the leds map?
					// if it's "turn off" assume that we already did that unless it's "force"
					if msg.mode != modeOff {
						log.Printf("Received led message: %v", msg)
						leds[msg.pin] = setLEDEffect(msg)
					}
				}
			default:
				keepReading = false
			}
		}
		// for anything that we're doing blink on, see if it's time to toggle
		// also anything that is modeUnset needs to be initiated
		now := rt.clock.Now()
		for i, v := range leds {
			// negative duration is "ignore"
			if v.duration < 0 {
				continue
			}

			if v.curMode == modeUnset {
				// transform broader categories of mode to on/off
				if v.mode == modeOff {
					rt.led.off(v.pin)
					v.curMode = modeOff
				} else {
					rt.led.on(v.pin)
					v.curMode = modeOn
				}
				v.lastUpdate = now
				v.startTime = v.lastUpdate
				// if it's just "off" or "on" set the duration to -1 so we never re-check
				if v.mode == modeOff {
					v.duration = -1
				}
				leds[i] = v
				continue
			}

			// duration expired means turn it off
			if v.duration > 0 && now.Sub(v.startTime) >= v.duration {
				if v.curMode != modeOff {
					rt.led.off(v.pin)
				}
				// negative duration is expired
				// TODO: remove from the map to make processing faster
				v.duration = -1
				v.curMode = modeOff
				v.lastUpdate = time.Time{}
				v.startTime = time.Time{}
				leds[i] = v
				continue
			}

			timeInState := now.Sub(v.lastUpdate)
			var upTime, downTime time.Duration

			switch v.mode {
			case modeBlink10:
				upTime = 900
			case modeBlink25:
				upTime = 750
			case modeBlink50:
				upTime = 500
			case modeBlink75:
				upTime = 250
			case modeBlink90:
				upTime = 100
			case modeOn:
				upTime = 1000
			case modeOff:
			default:
				// nothing to do
				continue
			}

			downTime = 1000 - upTime

			if v.curMode == modeOff {
				if timeInState >= downTime*time.Millisecond {
					rt.led.on(v.pin)
					v.curMode = modeOn
					v.lastUpdate = now
					leds[i] = v
				}
			} else {
				if upTime < 1000 && timeInState >= upTime*time.Millisecond {
					rt.led.off(v.pin)
					v.curMode = modeOff
					v.lastUpdate = now
					leds[i] = v
				}
			}
		}

		// sleep for a bit (1/10s is our lowest resolution)
		rt.clock.Sleep(dLEDSleep)
	}
}
