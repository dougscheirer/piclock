package main

import (
	"log"

	// gpio lib
	"github.com/stianeikeland/go-rpio"
	// keyboard for sim mode
)

type rpioButtons struct {
	buttons map[string]button
}

func init() {
	features = append(features, "rpio-buttons")
}

func (rb *rpioButtons) getButtons() *map[string]button {
	return &rb.buttons
}

func (rb *rpioButtons) setupPinButtons(pins map[string]buttonMap, runtime runtimeConfig) error {
	rb.buttons = make(map[string]button)

	// map pins to buttons
	err := rpio.Open()
	if err != nil {
		log.Println(err.Error())
		return err
	}

	now := runtime.rtc.Now()

	for k, v := range pins {
		// TODO: configurable pin numbers and high or low
		// picking GPIO 4 results in collisions with I2C operations
		var btn button
		btn.button = v
		btn.pin = rpio.Pin(v.pin)

		// for now we only care about the "low" state
		btn.pin.Input()  // Input mode
		btn.pin.PullUp() // GND => button press

		btn.state = pressState{pressed: false, start: now, count: 0, changed: false}
		rb.buttons[k] = btn
	}

	return nil
}

func (rb *rpioButtons) setupButtons(pins map[string]buttonMap, runtime runtimeConfig) error {
	return rb.setupPinButtons(pins, runtime)
}

func (rb *rpioButtons) initButtons(settings configSettings) error {
	// nothing to init for GPIO buttons
	return nil
}

func (rb *rpioButtons) closeButtons() {
	// N/A, nothing special
}

func (rb *rpioButtons) readButtons(runtime runtimeConfig) (map[string]rpio.State, error) {
	ret := make(map[string]rpio.State)
	for k, v := range rb.buttons {
		ret[k] = v.pin.Read() // Read state from pin (High / Low)
	}

	return ret, nil
}
