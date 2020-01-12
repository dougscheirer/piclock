// +build !noleds

package main

import (
	"github.com/stianeikeland/go-rpio"
)

func errorLED(on bool) {
	pin := rpio.Pin(16)
	if on {
		pin.High()
	} else {
		pin.Low()
	}
}
