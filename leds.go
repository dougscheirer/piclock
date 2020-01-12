// +build !noleds

package main

import (
	"log"

	"github.com/stianeikeland/go-rpio"
)

func errorLED(on bool) {
	log.Printf("Set pin 16 to %v", on)
	pin := rpio.Pin(16)
	pin.Output()
	if on {
		pin.High()
	} else {
		pin.Low()
	}
}