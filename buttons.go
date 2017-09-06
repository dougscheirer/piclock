package main

import (
  "time"
  "math/rand"
  "log"
  // gpio lib
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
  pinNum    int       // number of GPIO pin
  pin       rpio.Pin  // rpio pin
  state     PressState
}

func simMainButton(cE chan Effect) {
  buttonCount := 0
  for true {
    // randomly pause between button mash
    time.Sleep(time.Duration(rand.Intn(30)) * time.Second)
    pressTime := time.Now()
    cE <- mainButton(true, 0)
    buttonCount++
    // one in 10 chance of a long hold
    if buttonCount % 10 == 3 {
      for i:=0;i<6;i++ {
        time.Sleep(time.Second)
        cE <- mainButton(true, time.Now().Sub(pressTime))
      }
    } else {
      time.Sleep(time.Second)
    }
    cE <- mainButton(false, time.Now().Sub(pressTime))
  }
}

func setupButtons(pins []int) ([]Button, error) {
  // map pins to buttons
  err := rpio.Open()
  if err != nil {
    log.Println(err.Error())
    return []Button{}, err
  }

  ret := make([]Button, len(pins))
  now := time.Now()

  for i:=0;i<len(pins);i++ {
    // TODO: configurable pin numbers and high or low
    // picking GPIO 4 results in collisions with I2C operations
    ret[i].pinNum = pins[i]
    ret[i].pin    = rpio.Pin(pins[i])

    // for now we only care about the "low" state
    ret[i].pin.Input()        // Input mode
    ret[i].pin.PullUp()       // GND => button press

    ret[i].state = PressState{pressed: false, start: now, count: 0, changed: false}
  }

  return ret, nil
}

// returns new array with all of the button data
func checkButtons(btns []Button) ([]Button) {
  now := time.Now()

  ret := make([]Button, len(btns))

  for i:=0;i<len(btns);i++ {
    res := btns[i].pin.Read()  // Read state from pin (High / Low)
    ret[i] = btns[i]
    ret[i].state.changed = false

    // 0 is pressed, we're GNDing the button (pullup mode)
    if res == 0 {
      // is this a change from before?
      if ret[i].state.pressed {
        // no button state change, update the duration count
        ret[i].state.count = int(now.Sub(ret[i].state.start) / time.Second)
        if btns[i].state.count != ret[i].state.count {
          ret[i].state.changed = true
	  log.Printf("Button changed state: %+v", ret[i])
        }
      } else {
        // just noticed it was pressed
       ret[i].state=PressState{pressed: true, start: now, count: 0, changed: true}
       log.Printf("Button changed state: %+v", ret[i])
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
        log.Printf("Button changed state: %+v", ret[i])
        }*/
      } else {
        // just noticed the release
        ret[i].state=PressState{pressed: false, start: now, count: 0, changed: true}
         log.Printf("Button changed state: %+v", ret[i])
     }
    }
  }

  return ret
}

func watchButtons(settings *Settings, cE chan Effect) {
  defer wg.Done()

  simulated := settings.GetString("button_simulated")

  if len(simulated) != 0 {
    simMainButton(cE)
  } else {
    pins := []int{25, 24}
    // 25 -> main button
    // 24 -> some other button
    buttons, err := setupButtons(pins)
    if err != nil {
      log.Println(err.Error())
      return
    }

    for true {
      newButtons := checkButtons(buttons)
      for i:=0;i<len(newButtons);i++ {
        if newButtons[i].state.changed {
          diff := time.Duration(newButtons[i].state.count) * time.Second
          switch pins[i] {
              case 25:
                cE <- mainButton(newButtons[i].state.pressed, diff)
              default:
                log.Printf("Unhandled button %d", pins[i])
            }
        }
      }

      buttons = newButtons
      time.Sleep(10*time.Millisecond)
    }
  }
}
