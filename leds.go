// +build !notapi

package main

import (
	"log"

	"github.com/stianeikeland/go-rpio"
)

func init() {
	features = append(features, "leds")

	err := rpio.Open()
	if err != nil {
		log.Fatalf(err.Error())
	}
}

func setLED(pinNum int, on bool) {
	log.Printf("Set pin %v to %v", pinNum, on)
	pin := rpio.Pin(pinNum)
	pin.Output()
	if on {
		pin.High()
	} else {
		pin.Low()
	}
}
