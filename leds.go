// +build !noleds

package main

import (
	"log"

	"github.com/stianeikeland/go-rpio"
)

func init() {
	features = append(features, "leds")
}

func setLED(pinNum int, on bool) {
	log.Printf("Set pin 16 to %v", on)
	pin := rpio.Pin(pinNum)
	pin.Output()
	if on {
		pin.High()
	} else {
		pin.Low()
	}
}
