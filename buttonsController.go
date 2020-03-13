package main

import (
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
	rpin   rpio.Pin
	state  pressState
}

const (
	btnDown = 0
	btnUp   = 1
)

func init() {
	// for runWatchButtons
	wg.Add(1)
}

func checkButtons(rt runtimeConfig) (map[string]button, error) {
	now := rt.clock.Now()
	ret := make(map[string]button)

	btns := rt.buttons.getButtons()
	results, err := rt.buttons.readButtons(rt)
	if err != nil {
		return ret, err
	}

	for k, v := range *btns {
		var res rpio.State = results[k]

		btn := v
		btn.state.changed = false

		// interpret the high/low state into btnUp or btnDown
		// based on the pullup value
		var btnState int
		if v.button.pullup {
			// 0 is pressed, 1 is not
			if res == rpio.High {
				btnState = btnUp
			} else {
				btnState = btnDown
			}
		} else {
			// 1 is pressed, 0 is not
			if res == rpio.Low {
				btnState = btnUp
			} else {
				btnState = btnDown
			}
		}

		if btnState == btnDown {
			// is this a change from before?
			if btn.state.pressed {
				// no button state change, update the duration count
				btn.state.count = int(now.Sub(btn.state.start) / time.Second)
				if (*btns)[k].state.count != btn.state.count {
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
			rt.logger.Printf("button changed state: %+v", btn)
		}
		(*btns)[k] = btn
	}

	return *btns, nil
}

func startWatchButtons(rt runtimeConfig) {
	rt.logger = &ThreadLogger{name: "Buttons"}
	go runWatchButtons(rt)
}

func runWatchButtons(rt runtimeConfig) {
	defer wg.Done()
	defer func() {
		rt.logger.Println("exiting runWatchButtons")
	}()

	settings := rt.settings
	comms := rt.comms
	err := rt.buttons.initButtons(settings)
	if err != nil {
		rt.logger.Println(err.Error())
		return
	}

	// we now should defer the closeButtons call to when this function exists
	defer rt.buttons.closeButtons()

	pins := make(map[string]buttonMap)
	pins[sMainBtn] = settings.GetButtonMap(sMainBtn)
	pins[sLongBtn] = settings.GetButtonMap(sLongBtn)
	pins[sDblBtn] = settings.GetButtonMap(sDblBtn)

	err = rt.buttons.setupButtons(pins, rt)
	if err != nil {
		rt.logger.Println(err.Error())
		return
	}

	for true {
		select {
		case <-comms.quit:
			// we shouldn't get here ATM
			rt.logger.Println("quit from runWatchButtons (surprise)")
			return
		default:
		}

		newButtons, err := checkButtons(rt)
		if err != nil {
			// we're done
			rt.logger.Println("quit from runWatchButtons")
			close(comms.quit)
			return
		}

		for k, v := range newButtons {
			if v.state.changed {
				diff := time.Duration(v.state.count) * time.Second
				switch k {
				case sMainBtn:
					rt.logger.Println("sending main button message")
					comms.effects <- mainButtonEffect(v.state.pressed, diff)
					comms.chkAlarms <- mainButtonAlmMsg(v.state.pressed, diff)
				case sLongBtn:
					rt.logger.Println("sending long button messages")
					comms.effects <- longButtonEffect(v.state.pressed, diff)
					comms.chkAlarms <- longButtonAlmMsg(v.state.pressed, diff)
				case sDblBtn:
					rt.logger.Println("sending double click button message")
					comms.effects <- doubleButtonEffect(v.state.pressed, diff)
					comms.chkAlarms <- doubleButtonAlmMsg(v.state.pressed, diff)
				default:
					rt.logger.Printf("Unhandled button %s", k)
				}
			}
		}

		rt.clock.Sleep(dButtonSleep)
	}
}
