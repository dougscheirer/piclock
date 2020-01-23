package main

import (
	"github.com/stianeikeland/go-rpio"
)

type noButtons struct {
	buttons map[string]button
}

func (nb *noButtons) getButtons() *map[string]button {
	return &nb.buttons
}

func (nb *noButtons) readButtons(runtime runtimeConfig) (map[string]rpio.State, error) {
	ret := make(map[string]rpio.State)
	down := false
	for k := range nb.buttons {
		if !down {
			ret[k] = btnUp
		} else {
			ret[k] = btnDown
		}
	}
	return ret, nil
}

func (nb *noButtons) setupButtons(pins map[string]buttonMap, runtime runtimeConfig) error {
	nb.buttons = make(map[string]button)
	for k, v := range pins {
		nb.buttons[k] = button{button: v}
	}
	return nil
}

func (nb *noButtons) initButtons(settings configSettings) error {
	return nil
}

func (nb *noButtons) closeButtons() {
}
