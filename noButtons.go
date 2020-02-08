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
		// default this to "not pressed"
		// for pullup, GND is pressed, +V is open
		// for !pullup, +V is pressed, GND is open
		if v.pullup {
			nb.states[k] = rpio.High
		} else {
			nb.states[k] = rpio.Low
		}
	}
	return nil
}

func (nb *noButtons) initButtons(settings configSettings) error {
	return nil
}

func (nb *noButtons) closeButtons() {
}

func (nb *noButtons) setStates(btns map[string]rpio.State) {
	for k, v := range btns {
		nb.states[k] = v
	}
}

func (nb *noButtons) clear() {
	for k := range nb.buttons {
		// for pullup buttons +V is not pressed
		if nb.buttons[k].button.pullup {
			nb.states[k] = rpio.High
		} else {
			nb.states[k] = rpio.Low
		}
	}
}
