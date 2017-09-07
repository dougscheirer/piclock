package main

import (
  "time"
  "log"
  "errors"
  // gpio lib
  "github.com/stianeikeland/go-rpio"
  // keyboard for sim mode
  "github.com/nsf/termbox-go"
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

const (
    BTN_DOWN = 0    // 0 is pressed, we're GNDing the button (pullup mode)
    BTN_UP = 1
)

func simSetupButtons(pins []int, buttonMap string) []Button {
  // return a list of buttons with the char as the "pin num"
  ret := make([]Button, len(pins))
  now := time.Now()

  for i:=0;i<len(ret);i++ {
    if i >= len(buttonMap) {
      log.Printf("No key map for %v", pins[i])
      ret[i].pinNum = -1
      ret[i].state = PressState{pressed: false, start: now, count: 0, changed: false}
      continue
    }
    log.Printf("Key map for pin %d is %c", pins[i], buttonMap[i])
    ret[i].pinNum = int(buttonMap[i])
    ret[i].state = PressState{pressed: false, start: now, count: 0, changed: false}
  }
  return ret
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

func checkKeyboard(btns []Button) ([]rpio.State, error) {
  ret := make([]rpio.State, len(btns))

  // poll with quick timeout
  // no key means "no change"
  go func() {
    time.Sleep(100*time.Millisecond)
    termbox.Interrupt()
  }()

  var ev termbox.Event
  waitForInterrupt := true
  for waitForInterrupt {
    evTemp := termbox.PollEvent();
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
  for i:=0;i<len(ret);i++ {
    match := btns[i].pinNum == int(ev.Ch)
    if btns[i].state.pressed {
      // orig state is down
      if match {
        ret[i] = BTN_UP
      } else {
        // orig state is up
        ret[i] = BTN_DOWN
      }
    } else {
      if match {
        ret[i] = BTN_DOWN
      } else {
        // orig state is up
        ret[i] = BTN_UP
      }
    }
  }

  return ret, nil
}

// returns new array with all of the button data
func checkButtons(btns []Button, sim bool) ([]Button, error) {
  now := time.Now()
  ret := make([]Button, len(btns))

  var simResults []rpio.State
  var err error

  // simulated mode we check it all at once or we wait a lot
  if sim {
    simResults, err = checkKeyboard(btns)
    if err != nil {
      return btns, err
    }
  }

  for i:=0;i<len(btns);i++ {
    var res rpio.State
    if sim {
      res = simResults[i]
    } else {
      res = btns[i].pin.Read()  // Read state from pin (High / Low)
    }

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
        ret[i].state=PressState{pressed: true, start: now, count: 0, changed: true}
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
        ret[i].state=PressState{pressed: false, start: now, count: 0, changed: true}
     }
    }
    if ret[i].state.changed {
      log.Printf("Button changed state: %+v", ret[i])
    }
  }

  return ret, nil
}

func runWatchButtons(settings *Settings, quit chan struct{}, cE chan Effect) {
  defer wg.Done()

  simulated := settings.GetString("button_simulated")
  sim := false

  if len(simulated) > 0 {
    sim = true

    err := termbox.Init()
    if err != nil {
      panic(err)
    }

    termbox.SetInputMode(termbox.InputEsc | termbox.InputMouse)
    termbox.Flush()

    // close it later
    defer termbox.Close()
  }

  var buttons []Button
  pins := []int{25, 24}
  // 25 -> main button
  // 24 -> some other button

  if sim {
    buttons = simSetupButtons(pins, simulated)
  } else {
    var err error
    buttons, err = setupButtons(pins)
    if err != nil {
      log.Println(err.Error())
      return
    }
  }

  for true {
    select {
    case <- quit:
      // we shouldn't get here ATM
      log.Println("quit from runWatchButtons")
      return
    default:
    }

    newButtons, err := checkButtons(buttons, sim)
    if err != nil {
      // we're done
      log.Println("quit from runWatchButtons")
      close(quit)
      return
    }

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
