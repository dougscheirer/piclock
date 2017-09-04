package main

import (
  "time"
  "math/rand"
  // gpio lib
  "github.com/stianeikeland/go-rpio"
)

type ButtonInfo struct {
  pressed   bool
  duration  time.Duration
}

func watchButtons(settings *Settings, cE chan Effect) {
  defer wg.Done()

  simulated := settings.GetString("button_simulated")

  buttonCount := 0
  for true {
    if len(simulated) != 0 {
      // randomly pause between button mash
      time.Sleep(time.Duration(rand.Intn(30)) * time.Second)
      pressTime := time.Now()
      cE <- mainButtonPressed(0)
      buttonCount++
      // one in 10 chance of a long hold
      if buttonCount % 10 == 3 {
        for i:=0;i<6;i++ {
          time.Sleep(time.Second)
          cE <- mainButtonPressed(time.Now().Sub(pressTime))
        }
      } else {
        time.Sleep(time.Second)
      }
      cE <- mainButtonReleased(time.Now().Sub(pressTime))
    } else {
      // map ports to buttons
      err := rpio.Open()
      if err != nil {
        logMessage(err.Error())
        return
      }

      // TODO: configurable pin numbers and high or low
      // picking GPIO 4 results in collisions with I2C operations
      pin := rpio.Pin(25)

      // for now we only care about the "low" state
      pin.Input()        // Input mode
      pin.PullUp()       // GND => button press
      pressed := false
      pressTime := time.Now()
      // send "the button is still pressed" message every 1 sec
      lastMsg := pressTime

      for true {
        res := pin.Read()  // Read state from pin (High / Low)
        now := time.Now()
        if !pressed {
          if res == 0 {
            pressTime = now
            lastMsg = now
            cE <- mainButtonPressed(0)
            buttonCount++
            pressed = true
          }
        } else {
          if res == 1 {
            cE <- mainButtonReleased(time.Now().Sub(pressTime))
            pressed = false
          } else if now.Sub(lastMsg) > time.Second {
            lastMsg = now
            cE <- mainButtonPressed(now.Sub(pressTime))
          }
        }
        time.Sleep(10*time.Millisecond)
      }
    }
  }
}
