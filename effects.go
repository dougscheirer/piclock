package main

import (
  "piclock/sevenseg_backpack"
  "time"
  "fmt"
  "errors"
)

type Effect struct {
  id string  // TODO: a struct to tell the effects generator what to do
  val interface{}
}

// channel messagimng functions
func mainButtonPressed(d time.Duration) Effect {
  return Effect{ id:"mainButton", val : ButtonInfo{pressed: true, duration: d}  }
}

func mainButtonReleased(d time.Duration) Effect {
  return Effect{ id:"mainButton", val : ButtonInfo{pressed: false, duration: d} }
}

func setCountdownMode(alarm Alarm) Effect {
  return Effect{id:"countdown", val: alarm}
}

func setAlarmMode(alarm Alarm) Effect {
  return Effect{id:"alarm", val: alarm}
}

func alarmError(d time.Duration) Effect {
  return Effect{ id: "alarmError", val: d }
}

func toggleDebugDump(on bool) Effect {
  return Effect{ id: "debug", val: on }
}

func printEffect(s string) Effect {
  return Effect{ id: "print", val: s }
}

func replaceAtIndex(in string, r rune, i int) string {
  out := []rune(in)
  out[i] = r
  return string(out)
}

func toButtonInfo(val interface{}) (*ButtonInfo, error) {
  switch v:=val.(type) {
  case ButtonInfo:
    return &v, nil
  default:
    return nil, errors.New(fmt.Sprintf("Bad type: %T", v))
  }
}

func toBool(val interface{}) (bool, error) {
  switch v := val.(type) {
    case bool:
      return v, nil
    default:
      return false, errors.New(fmt.Sprintf("Bad type: %T", v))
  }
}

func toInt(val interface{}) (int, error) {
  switch v := val.(type) {
    case int:
      return v, nil
    default:
      return -1, errors.New(fmt.Sprintf("Bad type: %T", v))
  }
}

func toAlarm(val interface{}) (*Alarm, error) {
  switch v := val.(type) {
  case Alarm:
    return &v, nil
  default:
    return nil, errors.New(fmt.Sprintf("Bad type: %T", v))
  }
}

func toString(val interface{}) (string, error) {
  switch v := val.(type) {
  case string:
    return v, nil
  default:
    return "", errors.New(fmt.Sprintf("Bad type: %T", v))
  }
}

func toDuration(val interface{}) (time.Duration, error) {
  switch v := val.(type) {
  case time.Duration:
    return v, nil
  default:
    return 0, errors.New(fmt.Sprintf("Bad type: %T", v))
  }
}

func displayClock(display *sevenseg_backpack.Sevenseg) {
  // standard time display
  colon := "15:04"
  now := time.Now()
  if now.Second() % 2 == 0 {
    // no space required for the colon
    colon = "1504"
  }

  timeString := now.Format(colon)
  if timeString[0] == '0' {
    timeString = replaceAtIndex(timeString, ' ', 0)
  }

  err := display.Print(timeString)
  if err != nil {
    fmt.Printf("Error: %s\n", err.Error())
  }
}

func displayCountdown(display *sevenseg_backpack.Sevenseg, alarm *Alarm) bool {
  // calculate 10ths of secs to alarm time
  count := alarm.When.Sub(time.Now()) / (time.Second/10)
  if count > 9999 {
    count = 9999
  } else if count <= 0 {
    return false
  }
  s := fmt.Sprintf("%d.%d", count / 10, count % 10)
  var blinkRate uint8 = sevenseg_backpack.BLINK_OFF
  if count < 100 {
    blinkRate = sevenseg_backpack.BLINK_2HZ
  }
  display.SetBlinkRate(blinkRate)
  display.Print(s)
  return true
}

func runEffects(settings *Settings, cE chan Effect, cL chan LoaderMsg) {
  defer wg.Done()

  display, err := sevenseg_backpack.Open(
    settings.GetByte("i2c_device"),
    settings.GetInt("i2c_bus"),
    settings.GetBool("i2c_simulated"))

  if err != nil {
    fmt.Printf("Error: %s", err.Error())
    return
  }

  // turn on LED dump?
  display.DebugDump(settings.GetBool("debug_dump"))

  display.SetBrightness(3)
  // ready to rock
  display.DisplayOn(true)

  var mode string = "clock"
  var countdown *Alarm
  var error_id = 0
  alarmSegment := 0
  DEFAULT_SLEEP := settings.GetDuration("sleepTime")
  sleepTime := DEFAULT_SLEEP

  for true {
    var e Effect
    select {
    case e = <- cE:
      switch e.id {
        case "debug":
          v, _ := toBool(e.val)
          display.DebugDump(v)
        case "clock":
          mode = e.id
        case "countdown":
          mode = e.id
          countdown, _ = toAlarm(e.val)
          sleepTime = 10 * time.Millisecond
        case "alarmError":
          // TODO: alarm error LED
          display.Print("Err")
          d, _ := toDuration(e.val)
          time.Sleep(d)
        case "terminate":
          fmt.Printf("terminate")
          return
        case "print":
          v, _ := toString(e.val)
          display.Print(v)
          time.Sleep(time.Second)
        case "alarm":
          mode = e.id
          alm, _ := toAlarm(e.val)
          sleepTime = 10*time.Millisecond
          fmt.Printf(">>>>>>>>>>>>>>> ALARM <<<<<<<<<<<<<<<<<<\n%s %s %s\n", alm.Name, alm.When, alm.Effect)
        case "mainButton":
          info, _ := toButtonInfo(e.val)
          if info.pressed {
            logMessage("Main button pressed")
            switch mode {
              case "alarm":
                // TODO: cancel the alarm
                mode = "clock"
                sleepTime = DEFAULT_SLEEP
              case "countdown":
                // cancel the alarm
                mode = "clock"
                cL <- handledMessage(*countdown)
                countdown = nil
              case "clock":
                if info.duration > 5 * time.Second {
                  cL <- reloadMessage()
                }
              default:
                logMessage(fmt.Sprintf("No action for mode %s", mode))
            }
          } else {
            logMessage(fmt.Sprintf("Main button released: %dms", info.duration / time.Millisecond))
          }
        default:
          fmt.Printf("Unhandled %s\n", e.id)
      }
    default:
      // nothing?
      time.Sleep(time.Duration(sleepTime))
    }

    switch mode {
      case "clock":
        displayClock(display)
      case "countdown":
        if !displayCountdown(display, countdown) {
          mode = "clock"
          sleepTime = DEFAULT_SLEEP
        }
      case "alarmError":
        fmt.Sprintf("Error: %d\n", error_id)
        display.Print("Err")
      case "output":
        // do nothing
      case "alarm":
        // do a strobing 0, light up segments 0 - 5
        display.RefreshOn(false)
        display.SetBlinkRate(sevenseg_backpack.BLINK_OFF)
        display.ClearDisplay()
        for i:=0;i<4;i++ {
          display.SegmentOn(byte(i), byte(alarmSegment), true)
        }
        display.RefreshOn(true)
        alarmSegment = (alarmSegment + 1) % 6
      default:
        logMessage(fmt.Sprintf("Unknown mode: '%s'\n", mode))
    }
  }

  display.DisplayOn(false)
}
