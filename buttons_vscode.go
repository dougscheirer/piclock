// +build nobuttons

package main

import (
	"github.com/stianeikeland/go-rpio"
)

func init() {
	features = append(features, "no-buttons")
}

func readButtons(runtime runtimeConfig, btns map[string]button) (map[string]rpio.State, error) {
	// simulated mode we check it all at once or we wait a lot
	ret := make(map[string]rpio.State)
	for k, _ := range btns {
		ret[k] = btnUp
	}
	return ret, nil
}

func setupButtons(pins map[string]buttonMap, settings *configSettings, runtime runtimeConfig) (map[string]button, error) {
	return make(map[string]button), nil
}

func initButtons(settings *configSettings) error {
	return nil
}

func closeButtons() {
}
