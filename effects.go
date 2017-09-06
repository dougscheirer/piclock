package main

import (
  "piclock/sevenseg_backpack"
  "time"
  "fmt"

  "errors"
  "log"
)

type Effect struct {
  id string  // TODO: a struct to tell the effects generator what to do
  val interface{}
}

type Print struct {
  s string
  d time.Duration
}

type ButtonInfo struct {
  pressed   bool
  duration  time.Duration
}

// channel messagimng functions
func mainButton(p bool, d time.Duration) Effect {
  return Effect{ id:"mainButton", val : ButtonInfo{pressed: p, duration: d}  }
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

func printEffect(s string, d time.Duration) Effect {
  return Effect{ id: "print", val: Print{s:s, d:d} }
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

func toPrint(val interface{}) (*Print, error) {
  switch v := val.(type) {
  case Print:
    return &v, nil
  default:
    return nil, errors.New(fmt.Sprintf("Bad type: %T", v))
  }
}

func displayClock(display *sevenseg_backpack.Sevenseg, dot bool) {
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
  if dot {
    timeString+="."
  }
  err := display.Print(timeString)
  if err != nil {
    log.Printf("Error: %s\n", err.Error())
  }
}

func displayCountdown(display *sevenseg_backpack.Sevenseg, alarm *Alarm, dot bool) bool {
  // calculate 10ths of secs to alarm time
  count := alarm.When.Sub(time.Now()) / (time.Second/10)
  if count > 9999 {
    count = 9999
  } else if count <= 0 {
    return false
  }
  s := fmt.Sprintf("%d.%d", count / 10, count % 10)
  if dot {
    s+="."
  }
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
    log.Printf("Error: %s", err.Error())
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
  buttonPressActed := false
  buttonDot := false

  for true {
    var e Effect

    skip := false

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
          log.Println("terminate")
          return
        case "print":
          v, _ := toPrint(e.val)
          log.Printf("Print: %s (%d)", v.s, v.d)
          display.Print(v.s)
          time.Sleep(v.d)
          skip = true // don't immediately print the clock in clock mode
        case "alarm":
          mode = e.id
          alm, _ := toAlarm(e.val)
          sleepTime = 10*time.Millisecond
          log.Printf(">>>>>>>>>>>>>>> ALARM <<<<<<<<<<<<<<<<<<\n%s %s %s\n", alm.Name, alm.When, alm.Effect)
        case "mainButton":
          info, _ := toButtonInfo(e.val)
          buttonDot = info.pressed
          if info.pressed {
            if buttonPressActed {
              log.Println("Ignore button hold")
            } else {
              log.Printf("Main button pressed: %dms", info.duration)
              switch mode {
                case "alarm":
                  // TODO: cancel the alarm
                  mode = "clock"
                  sleepTime = DEFAULT_SLEEP
                  buttonPressActed = true
                case "countdown":
                  // cancel the alarm
                  mode = "clock"
                  cL <- handledMessage(*countdown)
                  countdown = nil
                  buttonPressActed = true
                case "clock":
                  // more than 5 seconds is "reload"
                  if info.duration > 4 * time.Second {
                    cL <- reloadMessage()
                    buttonPressActed = true
                  }
                default:
                  log.Printf("No action for mode %s", mode)
              }
            }
          } else {
            buttonPressActed = false
            log.Printf("Main button released: %dms", info.duration / time.Millisecond)
          }
        default:
          log.Printf("Unhandled %s\n", e.id)
      }
    default:
      // nothing?
      time.Sleep(time.Duration(sleepTime))
    }

    // skip the mode stuff?
    if skip {
      continue
    }

    switch mode {
      case "clock":
        displayClock(display, buttonDot)
      case "countdown":
        if !displayCountdown(display, countdown, buttonDot) {
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
        log.Printf("Unknown mode: '%s'\n", mode)
    }
  }

  display.DisplayOn(false)
}
