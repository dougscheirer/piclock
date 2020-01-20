// +build nobuttons

package main

import (
	"github.com/stianeikeland/go-rpio"
)

func init() {
	features = append(features, "no-buttons")
}

func readButtons(btns []button) ([]rpio.State, error) {
	// simulated mode we check it all at once or we wait a lot
	ret := make([]rpio.State, len(btns))
	for i := 0; i < len(ret); i++ {
		ret[i] = btnUp
	}
	return ret, nil
}

func setupButtons(pins []int, settings *configSettings, runtime runtimeConfig) ([]button, error) {
	return make([]button, len(pins)), nil
}

func initButtons(settings *configSettings) error {
	return nil
}

func closeButtons() {
}
