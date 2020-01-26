package main

import (
	"github.com/stianeikeland/go-rpio"
)

type noButtons struct {
	buttons map[string]button
	states  map[string]rpio.State
}

func (nb *noButtons) getButtons() *map[string]button {
	return &nb.buttons
}

func (nb *noButtons) readButtons(rt runtimeConfig) (map[string]rpio.State, error) {
	return nb.states, nil
}

func (nb *noButtons) setupButtons(pins map[string]buttonMap, rt runtimeConfig) error {
	nb.buttons = make(map[string]button)
	nb.states = make(map[string]rpio.State)

	for k, v := range pins {
		nb.buttons[k] = button{button: v}
		nb.states[k] = rpio.High // TODO: configurable high/low button press
	}
	return nil
}

func (nb *noButtons) initButtons(settings configSettings) error {
	return nil
}

func (nb *noButtons) closeButtons() {
}

func (nb *noButtons) set(btns map[string]rpio.State) {
	for k, v := range btns {
		nb.states[k] = v
	}
}

func (nb *noButtons) clear() {
	for k := range nb.buttons {
		nb.states[k] = rpio.High
	}
}
