package main

import (
	"log"
	"time"

	"github.com/stianeikeland/go-rpio"
)

// check the press state, and return the press state
type pressState struct {
	pressed bool      // is it pressed?
	start   time.Time // when did this state start?
	count   int       // # of whole seconds since it started
	changed bool      // did the above data change at all?
}

type button struct {
	button buttonMap
	pin    rpio.Pin // rpio pin
	state  pressState
}

const (
	btnDown = 0 // 0 is pressed, we're GNDing the button (pullup mode)
	btnUp   = 1
)

func init() {
	// for runWatchButtons
	wg.Add(1)
}

func checkButtons(btns map[string]button, runtime runtimeConfig) (map[string]button, error) {
	now := runtime.rtc.now()
	ret := make(map[string]button)

	var results map[string]rpio.State
	var err error

	results, err = readButtons(runtime, btns)
	if err != nil {
		return ret, err
	}

	for k, v := range btns {
		var res rpio.State = results[k]

		btn := v
		btn.state.changed = false

		if res == btnDown {
			// is this a change from before?
			if btn.state.pressed {
				// no button state change, update the duration count
				btn.state.count = int(now.Sub(btn.state.start) / time.Second)
				if btns[k].state.count != btn.state.count {
					btn.state.changed = true
				}
			} else {
				// just noticed it was pressed
				btn.state = pressState{pressed: true, start: now, count: 0, changed: true}
			}
		} else {
			// not pressed, is that a state change?
			if !btn.state.pressed {
				// no button state change, update the duration count?
				// keep this less chatty, a button that is continually not pressed is not a state change
				/*
				   ret[i].state.count = int(now.Sub(ret[i].state.start) / time.Second)
				   if btns[i].state.count != ret[i].state.count {
				   ret[i].state.changed = true
				   }*/
			} else {
				// just noticed the release
				btn.state = pressState{pressed: false, start: now, count: 0, changed: true}
			}
		}
		if btn.state.changed {
			log.Printf("button changed state: %+v", btn)
		}
		ret[k] = btn
	}

	return ret, nil
}

func runWatchButtons(settings *configSettings, runtime runtimeConfig) {
	defer wg.Done()
	defer func() {
		log.Println("exiting runWatchButtons")
	}()

	comms := runtime.comms
	err := initButtons(settings)
	if err != nil {
		log.Println(err.Error())
		return
	}

	// we now should defer the closeButtons call to when this function exists
	defer closeButtons()

	var buttons map[string]button
	pins := make(map[string]buttonMap)
	pins["mainButton"] = settings.GetButtonMap("mainButton")

	buttons, err = setupButtons(pins, settings, runtime)
	if err != nil {
		log.Println(err.Error())
		return
	}

	for true {
		select {
		case <-comms.quit:
			// we shouldn't get here ATM
			log.Println("quit from runWatchButtons (surprise)")
			return
		default:
		}

		newButtons, err := checkButtons(buttons, runtime)
		if err != nil {
			// we're done
			log.Println("quit from runWatchButtons")
			close(comms.quit)
			return
		}

		for k, v := range newButtons {
			if v.state.changed {
				diff := time.Duration(v.state.count) * time.Second
				switch k {
				case "mainButton":
					log.Println("sending main button to effects")
					comms.effects <- mainButton(v.state.pressed, diff)
				default:
					log.Printf("Unhandled button %s", k)
				}
			}
		}

		buttons = newButtons
		runtime.rtc.sleep(10 * time.Millisecond)
	}
}
