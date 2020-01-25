package main

import (
	"log"

	"github.com/stianeikeland/go-rpio"
)

type rpioLed struct {
}

func (rpi *rpioLed) init() {
	err := rpio.Open()
	if err != nil {
		log.Fatalf(err.Error())
	}
}

func (rpi *rpioLed) set(pinNum int, on bool) {
	log.Printf("Set pin %v to %v", pinNum, on)
	pin := rpio.Pin(pinNum)
	pin.Output()
	if on {
		pin.High()
	} else {
		pin.Low()
	}
}

func (rpi *rpioLed) on(pin int) {
	rpi.set(pin, true)
}

func (rpi *rpioLed) off(pin int) {
	rpi.set(pin, false)
}
