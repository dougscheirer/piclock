package main

import (
	"errors"
	"time"

	// keyboard for sim mode
	"github.com/nsf/termbox-go"
	"github.com/stianeikeland/go-rpio"
)

func init() {
	features = append(features, "key-buttons")
}

type simButtons struct {
	buttons map[string]button
}

func (sb *simButtons) getButtons() *map[string]button {
	return &sb.buttons
}

func (sb *simButtons) simSetupButtons(pins map[string]buttonMap, runtime runtimeConfig) error {
	sb.buttons = make(map[string]button)

	// return a list of buttons with the char as the "pin num"
	now := runtime.rtc.Now()

	for k, v := range pins {
		var btn button
		btn.button = v
		btn.state = pressState{pressed: false, start: now, count: 0, changed: false}
		sb.buttons[k] = btn
	}
	return nil
}

func (sb *simButtons) checkKeyboard(runtime runtimeConfig) (map[string]rpio.State, error) {
	ret := make(map[string]rpio.State)

	// poll with quick timeout
	// no key means "no change"
	go func() {
		runtime.rtc.Sleep(100 * time.Millisecond)
		termbox.Interrupt()
	}()

	var ev termbox.Event
	waitForInterrupt := true
	for waitForInterrupt {
		evTemp := termbox.PollEvent()
		switch evTemp.Type {
		case termbox.EventKey:
			// add an exit key
			if evTemp.Key == termbox.KeyCtrlC {
				return ret, errors.New("Exit termbox loop")
			}
			ev = evTemp
		// wait for the interrupt to fire
		default:
			waitForInterrupt = false
			// no keys
		}
	}

	termbox.Flush()

	// char is toggle (down to up or up to down)
	// neither letter is "do not change"
	for k, v := range sb.buttons {
		match := v.button.key[0] == byte(ev.Ch)
		if v.state.pressed {
			// orig state is down
			if match {
				ret[k] = btnUp
			} else {
				// orig state is up
				ret[k] = btnDown
			}
		} else {
			if match {
				ret[k] = btnDown
			} else {
				// orig state is up
				ret[k] = btnUp
			}
		}
	}

	return ret, nil
}

func (sb *simButtons) readButtons(runtime runtimeConfig) (map[string]rpio.State, error) {
	// simulated mode we check it all at once or we wait a lot
	return sb.checkKeyboard(runtime)
}

func (sb *simButtons) setupButtons(pins map[string]buttonMap, runtime runtimeConfig) error {
	return sb.simSetupButtons(pins, runtime)
}

func (sb *simButtons) initButtons(settings configSettings) error {
	err := termbox.Init()
	if err != nil {
		return err
	}

	termbox.SetInputMode(termbox.InputEsc | termbox.InputMouse)
	termbox.Flush()

	// close it later
	return nil
}

func (sb *simButtons) closeButtons() {
	termbox.Close()
}
