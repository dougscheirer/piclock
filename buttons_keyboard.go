// +build keybuttons

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

func simSetupButtons(pins map[string]buttonMap, runtime runtimeConfig) (map[string]button, error) {
	// return a list of buttons with the char as the "pin num"
	ret := make(map[string]button)
	now := runtime.rtc.Now()

	for k, v := range pins {
		var btn button
		btn.button = v
		btn.state = pressState{pressed: false, start: now, count: 0, changed: false}
		ret[k] = btn
	}
	return ret, nil
}

func checkKeyboard(runtime runtimeConfig, btns map[string]button) (map[string]rpio.State, error) {
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
	for k, v := range btns {
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

func readButtons(runtime runtimeConfig, btns map[string]button) (map[string]rpio.State, error) {
	// simulated mode we check it all at once or we wait a lot
	return checkKeyboard(runtime, btns)
}

func setupButtons(pins map[string]buttonMap, settings *configSettings, runtime runtimeConfig) (map[string]button, error) {
	var buttons map[string]button
	var err error

	buttons, err = simSetupButtons(pins, runtime)

	return buttons, err
}

func initButtons(settings *configSettings) error {
	err := termbox.Init()
	if err != nil {
		return err
	}

	termbox.SetInputMode(termbox.InputEsc | termbox.InputMouse)
	termbox.Flush()

	// close it later
	return nil
}

func closeButtons() {
	termbox.Close()
}
