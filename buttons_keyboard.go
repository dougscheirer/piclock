// +build keybuttons

package main

import (
	"errors"
	"log"
	"time"

	// keyboard for sim mode
	"github.com/nsf/termbox-go"
	"github.com/stianeikeland/go-rpio"
)

func init() {
	features = append(features, "key-buttons")
}

func simSetupButtons(pins []int, buttonMap string, runtime runtimeConfig) ([]button, error) {
	// return a list of buttons with the char as the "pin num"
	ret := make([]button, len(pins))
	now := runtime.rtc.now()

	for i := 0; i < len(ret); i++ {
		if i >= len(buttonMap) {
			log.Printf("No key map for %v", pins[i])
			ret[i].pinNum = -1
			ret[i].state = pressState{pressed: false, start: now, count: 0, changed: false}
			continue
		}
		log.Printf("Key map for pin %d is %c", pins[i], buttonMap[i])
		ret[i].pinNum = int(buttonMap[i])
		ret[i].state = pressState{pressed: false, start: now, count: 0, changed: false}
	}
	return ret, nil
}

func checkKeyboard(btns []button) ([]rpio.State, error) {
	ret := make([]rpio.State, len(btns))

	// poll with quick timeout
	// no key means "no change"
	go func() {
		time.Sleep(100 * time.Millisecond)
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
	for i := 0; i < len(ret); i++ {
		match := btns[i].pinNum == int(ev.Ch)
		if btns[i].state.pressed {
			// orig state is down
			if match {
				ret[i] = btnUp
			} else {
				// orig state is up
				ret[i] = btnDown
			}
		} else {
			if match {
				ret[i] = btnDown
			} else {
				// orig state is up
				ret[i] = btnUp
			}
		}
	}

	return ret, nil
}

func readButtons(btns []button) ([]rpio.State, error) {
	// simulated mode we check it all at once or we wait a lot
	return checkKeyboard(btns)
}

func setupButtons(pins []int, settings *settings, runtime runtimeConfig) ([]button, error) {
	var buttons []button
	var err error

	simulated := settings.GetString("button_simulated")
	buttons, err = simSetupButtons(pins, simulated, runtime)

	return buttons, err
}

func initButtons(settings *settings) error {
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
