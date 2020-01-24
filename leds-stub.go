package main

import "log"

type logLed struct {
	leds []bool
}

func (ll *logLed) init() {
	// log the init?
	ll.leds = make([]bool, 32)
	for i := range ll.leds {
		ll.leds[i] = false
	}
}

func (ll *logLed) set(pinNum int, on bool) {
	ll.leds[pinNum] = on
	log.Printf("Set LED %v to %v", pinNum, on)
}

func (ll *logLed) on(pinNum int) {
	ll.set(pinNum, true)
}

func (ll *logLed) off(pinNum int) {
	ll.set(pinNum, false)
}
