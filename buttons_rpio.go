// +build !nobuttons
// +build !keybuttons

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

func setupPinButtons(pins map[string]buttonMap, runtime runtimeConfig) (map[string]button, error) {
	// map pins to buttons
	err := rpio.Open()
	if err != nil {
		log.Println(err.Error())
		return map[string]button{}, err
	}

	ret := make(map[string]button, len(pins))
	now := runtime.rtc.now()

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
		ret[k] = btn
	}

	return ret, nil
}

func setupButtons(pins map[string]buttonMap, settings *configSettings, runtime runtimeConfig) (map[string]button, error) {
	return setupPinButtons(pins, runtime)
}

func initButtons(settings *configSettings) error {
	// nothing to init for GPIO buttons
	return nil
}

func closeButtons() {
	// N/A, nothing special
}

func readButtons(runtime runtimeConfig, btns map[string]button) (map[string]rpio.State, error) {
	ret := make(map[string]rpio.State)
	for k, v := range btns {
		ret[k] = v.pin.Read() // Read state from pin (High / Low)
	}

	return ret, nil
}
