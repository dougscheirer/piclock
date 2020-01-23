package main

import "log"

type logLed struct {
}

func init() {
	features = append(features, "noleds")
}

func (ll *logLed) init() {
	// log the init?
}

func (ll *logLed) set(pinNum int, on bool) {
	log.Printf("Set LED %v to %v", pinNum, on)
}

func (ll *logLed) on(pinNum int) {
	ll.set(pinNum, true)
}

func (ll *logLed) off(pinNum int) {
	ll.set(pinNum, false)
}
