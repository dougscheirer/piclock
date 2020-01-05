// +build !nobuttons

package main

import (
	"log"
	"time"

	// gpio lib
	"github.com/stianeikeland/go-rpio"
	// keyboard for sim mode
)

func setupPinButtons(pins []int) ([]Button, error) {
	// map pins to buttons
	err := rpio.Open()
	if err != nil {
		log.Println(err.Error())
		return []Button{}, err
	}

	ret := make([]Button, len(pins))
	now := time.Now()

	for i := 0; i < len(pins); i++ {
		// TODO: configurable pin numbers and high or low
		// picking GPIO 4 results in collisions with I2C operations
		ret[i].pinNum = pins[i]
		ret[i].pin = rpio.Pin(pins[i])

		// for now we only care about the "low" state
		ret[i].pin.Input()  // Input mode
		ret[i].pin.PullUp() // GND => button press

		ret[i].state = PressState{pressed: false, start: now, count: 0, changed: false}
	}

	return ret, nil
}

func setupButtons(settings *Settings) []Button {
	pins := []int{25, 24}
	// 25 -> main button
	// 24 -> some other button
	var err error
	var buttons []Button

	buttons, err = setupPinButtons(pins)
	if err != nil {
		log.Println(err.Error())
		return nil
	}

	return buttons
}
