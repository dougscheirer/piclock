// +build !nobuttons

package main

import (
	"log"

	// gpio lib
	"github.com/stianeikeland/go-rpio"
	// keyboard for sim mode
)

func init() {
	features = append(features, "rpio-buttons")
}

func setupPinButtons(pins []int, runtime runtimeConfig) ([]button, error) {
	// map pins to buttons
	err := rpio.Open()
	if err != nil {
		log.Println(err.Error())
		return []button{}, err
	}

	ret := make([]button, len(pins))
	now := runtime.rtc.now()

	for i := 0; i < len(pins); i++ {
		// TODO: configurable pin numbers and high or low
		// picking GPIO 4 results in collisions with I2C operations
		ret[i].pinNum = pins[i]
		ret[i].pin = rpio.Pin(pins[i])

		// for now we only care about the "low" state
		ret[i].pin.Input()  // Input mode
		ret[i].pin.PullUp() // GND => button press

		ret[i].state = pressState{pressed: false, start: now, count: 0, changed: false}
	}

	return ret, nil
}

func setupButtons(pins []int, settings *settings, runtime runtimeConfig) ([]button, error) {
	return setupPinButtons(pins, runtime)
}

func initButtons(settings *settings) error {
	// nothing to init for GPIO buttons
	return nil
}

func closeButtons() {
	// N/A, nothing special
}

func readButtons(btns []button) ([]rpio.State, error) {
	ret := make([]rpio.State, len(btns))
	for i := 0; i < len(btns); i++ {
		ret[i] = btns[i].pin.Read() // Read state from pin (High / Low)
	}

	return ret, nil
}
