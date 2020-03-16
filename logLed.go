package main

import (
	"fmt"
)

type logLed struct {
	leds       []bool
	audit      []string
	disableLog bool
	logger     flogger
}

func (ll *logLed) init() {
	// log the init?
	ll.leds = make([]bool, 32)
	ll.audit = make([]string, 0)

	for i := range ll.leds {
		ll.leds[i] = false
	}
	ll.logger = &ThreadLogger{name: "LEDs"}
}

func (ll *logLed) set(pinNum int, on bool) {
	ll.leds[pinNum] = on
	if !ll.disableLog {
		ll.logger.Printf("Set LED %v to %v", pinNum, on)
	}
	ll.audit = append(ll.audit, fmt.Sprintf("Set LED %v to %v", pinNum, on))
}

func (ll *logLed) on(pinNum int) {
	ll.set(pinNum, true)
}

func (ll *logLed) off(pinNum int) {
	ll.set(pinNum, false)
}
