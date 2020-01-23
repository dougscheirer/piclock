package main

import (
	"log"

	"github.com/stianeikeland/go-rpio"
)

type rpiLed struct {
}

func (rpi *rpiLed) init() {
	err := rpio.Open()
	if err != nil {
		log.Fatalf(err.Error())
	}
}

func (rpi *rpiLed) set(pinNum int, on bool) {
	log.Printf("Set pin %v to %v", pinNum, on)
	pin := rpio.Pin(pinNum)
	pin.Output()
	if on {
		pin.High()
	} else {
		pin.Low()
	}
}

func (rpi *rpiLed) on(pin int) {
	rpi.set(pin, true)
}

func (rpi *rpiLed) off(pin int) {
	rpi.set(pin, false)
}
