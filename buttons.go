package main

import (
  "time"
  "strings"
  "os"
  "bufio"
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

  for true {
    if len(simulated) != 0 {
        lSim := strings.ToLower(simulated)
        uSim := strings.ToUpper(simulated)

        // TODO: map buttons to keys
        reader := bufio.NewReader(os.Stdin)
        input, _ := reader.ReadString('\n')

        if []byte(input)[0] == lSim[0] {
          pressTime := time.Now()
          cE <- mainButtonPressed()
          time.Sleep(100 * time.Millisecond)
          cE <- mainButtonReleased(time.Now().Sub(pressTime))
        } else if []byte(input)[0] == uSim[0] {
          pressTime := time.Now()
          cE <- mainButtonPressed()
          time.Sleep(6 * time.Second)
          cE <- mainButtonReleased(time.Now().Sub(pressTime))
        }
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

        for true {
          res := pin.Read()  // Read state from pin (High / Low)
          if !pressed {
            if res == 0 {
              pressTime = time.Now()
              cE <- mainButtonPressed()
              pressed = true
            }
          } else {
            if res == 1 {
              cE <- mainButtonReleased(time.Now().Sub(pressTime))
              pressed = false
            }
          }
          time.Sleep(10*time.Millisecond)
        }
      }
  }
}
