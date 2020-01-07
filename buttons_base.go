package main

import (
	"log"
	"time"

	"github.com/stianeikeland/go-rpio"
)

// check the press state, and return the press state
type PressState struct {
	pressed bool      // is it pressed?
	start   time.Time // when did this state start?
	count   int       // # of whole seconds since it started
	changed bool      // did the above data change at all?
}

type Button struct {
	pinNum int      // number of GPIO pin
	pin    rpio.Pin // rpio pin
	state  PressState
}

const (
	BTN_DOWN = 0 // 0 is pressed, we're GNDing the button (pullup mode)
	BTN_UP   = 1
)

func checkButtons(btns []Button) ([]Button, error) {
	now := time.Now()
	ret := make([]Button, len(btns))

	var results []rpio.State
	var err error

	results, err = readButtons(btns)
	if err != nil {
		return ret, err
	}

	for i := 0; i < len(btns); i++ {
		var res rpio.State = results[i]

		ret[i] = btns[i]
		ret[i].state.changed = false

		if res == BTN_DOWN {
			// is this a change from before?
			if ret[i].state.pressed {
				// no button state change, update the duration count
				ret[i].state.count = int(now.Sub(ret[i].state.start) / time.Second)
				if btns[i].state.count != ret[i].state.count {
					ret[i].state.changed = true
				}
			} else {
				// just noticed it was pressed
				ret[i].state = PressState{pressed: true, start: now, count: 0, changed: true}
			}
		} else {
			// not pressed, is that a state change?
			if !ret[i].state.pressed {
				// no button state change, update the duration count?
				// keep this less chatty, a button that is continually not pressed is not a state change
				/*
				   ret[i].state.count = int(now.Sub(ret[i].state.start) / time.Second)
				   if btns[i].state.count != ret[i].state.count {
				   ret[i].state.changed = true
				   }*/
			} else {
				// just noticed the release
				ret[i].state = PressState{pressed: false, start: now, count: 0, changed: true}
			}
		}
		if ret[i].state.changed {
			log.Printf("Button changed state: %+v", ret[i])
		}
	}

	return ret, nil
}

func runWatchButtons(settings *Settings, comms CommChannels) {
	defer wg.Done()

	err := initButtons(settings)
	if err != nil {
		log.Println(err.Error())
		return
	}

	var buttons []Button
	pins := []int{25, 24}
	// 25 -> main button
	// 24 -> some other button

	buttons, err = setupButtons(pins, settings)
	if err != nil {
		log.Println(err.Error())
		return
	}

	for true {
		select {
		case <-comms.quit:
			// we shouldn't get here ATM
			log.Println("quit from runWatchButtons")
			return
		default:
		}

		newButtons, err := checkButtons(buttons)
		if err != nil {
			// we're done
			log.Println("quit from runWatchButtons")
			close(comms.quit)
			return
		}

		for i := 0; i < len(newButtons); i++ {
			if newButtons[i].state.changed {
				diff := time.Duration(newButtons[i].state.count) * time.Second
				switch pins[i] {
				case 25:
					comms.effects <- mainButton(newButtons[i].state.pressed, diff)
				default:
					log.Printf("Unhandled button %d", pins[i])
				}
			}
		}

		buttons = newButtons
		time.Sleep(10 * time.Millisecond)
	}
}
