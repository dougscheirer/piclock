package main

import (
  "piclock/sevenseg_backpack"
  "time"
  "fmt"
  "math/rand"
  "errors"
  "log"
  "path/filepath"
  "errors"
  "log"
  // audio player
  "github.com/gordonklaus/portaudio"
)

type Effect struct {
  id int
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

const (
  modeClock = iota
  modeAlarm
  modeAlarmError
  modeCountdown
  modeOutput
)

const (
  eClock = iota
  eDebug
  eMainButton
  eAlarmError
  eTerminate
  ePrint
  eAlarm
  eCountdown
)

// channel messaging functions
func mainButton(p bool, d time.Duration) Effect {
  return Effect{ id: eMainButton, val : ButtonInfo{pressed: p, duration: d}  }
}

func setCountdownMode(alarm Alarm) Effect {
  return Effect{id: eCountdown, val: alarm}
}

func setAlarmMode(alarm Alarm) Effect {
  return Effect{id:eAlarm, val: alarm}
}

func alarmError(d time.Duration) Effect {
  return Effect{ id: eAlarmError, val: d }
}

func toggleDebugDump(on bool) Effect {
  return Effect{ id: eDebug, val: on }
}

func printEffect(s string, d time.Duration) Effect {
  return Effect{ id: ePrint, val: Print{s:s, d:d} }
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

func displayClock(display *sevenseg_backpack.Sevenseg, blinkColon bool, dot bool) {
  // standard time display
  colon := "15:04"
  now := time.Now()
  if blinkColon && now.Second() % 2 == 0 {
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

func playAlarmEffect(settings *Settings, alm *Alarm, stop chan bool) {
  switch alm.Effect {
  case almMusic:
    PlayMP3(alm.Extra, true, stop)
    break
  case almFile:
    PlayMP3(alm.Extra, true, stop)
    break
  case almTones:
    playIt([]string{"250","340"}, []string{"100ms", "100ms", "100ms", "100ms", "100ms", "2000ms"}, stop)
    break
  default:
    // play a random mp3 in the cache
    s1 := rand.NewSource(time.Now().UnixNano())
    r1 := rand.New(s1)

    files, err := filepath.Glob(settings.GetString("musicPath") + "/*")
    if err != nil {
        log.Fatal(err)
        break
    }
    fname := files[r1.Intn(len(files))]
    log.Printf("Playing %s", fname)
    PlayMP3(fname, true, stop)
    break
  }
}

func stopAlarmEffect(stop chan bool) {
  stop <- true
}

func runEffects(settings *Settings, quit chan struct{}, cE chan Effect, cL chan LoaderMsg) {
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

  mode := modeClock
  var countdown *Alarm
  var error_id = 0
  alarmSegment := 0
  DEFAULT_SLEEP := settings.GetDuration("sleepTime")
  sleepTime := DEFAULT_SLEEP
  buttonPressActed := false
  buttonDot := false

  stopAlarm := make(chan bool, 1)

  for true {
    var e Effect

    skip := false

    select {
    case <- quit:
      log.Println("quit from runEffects")
      return
    case e = <- cE:
      switch e.id {
        case eDebug:
          v, _ := toBool(e.val)
          display.DebugDump(v)
        case eClock:
          mode = modeClock
        case eCountdown:
          mode = modeCountdown
          countdown, _ = toAlarm(e.val)
          sleepTime = 10 * time.Millisecond
        case eAlarmError:
          // TODO: alarm error LED
          display.Print("Err")
          d, _ := toDuration(e.val)
          time.Sleep(d)
        case eTerminate:
          log.Println("terminate")
          return
        case ePrint:
          v, _ := toPrint(e.val)
          log.Printf("Print: %s (%d)", v.s, v.d)
          display.Print(v.s)
          time.Sleep(v.d)
          skip = true // don't immediately print the clock in clock mode
        case eAlarm:
          mode = modeAlarm
          alm, _ := toAlarm(e.val)
          sleepTime = 10*time.Millisecond
          log.Printf(">>>>>>>>>>>>>>> ALARM <<<<<<<<<<<<<<<<<<")
          log.Printf("%s %s %d", alm.Name, alm.When, alm.Effect)
          playAlarmEffect(settings, alm, stopAlarm)
        case eMainButton:
          info, _ := toButtonInfo(e.val)
          buttonDot = info.pressed
          if info.pressed {
            if buttonPressActed {
              log.Println("Ignore button hold")
            } else {
              log.Printf("Main button pressed: %dms", info.duration)
              switch mode {
                case modeAlarm:
                  // cancel the alarm
                  mode = modeClock
                  sleepTime = DEFAULT_SLEEP
                  buttonPressActed = true
                  stopAlarmEffect(stopAlarm)
                case modeCountdown:
                  // cancel the alarm
                  mode = modeClock
                  cL <- handledMessage(*countdown)
                  countdown = nil
                  buttonPressActed = true
                case modeClock:
                  // more than 5 seconds is "reload"
                  if info.duration > 4 * time.Second {
                    cL <- reloadMessage()
                    buttonPressActed = true
                  }
                default:
                  log.Printf("No action for mode %d", mode)
              }
            }
          } else {
            buttonPressActed = false
            log.Printf("Main button released: %dms", info.duration / time.Millisecond)
          }
        default:
          log.Printf("Unhandled %d\n", e.id)
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
      case modeClock:
        displayClock(display, settings.GetBool("blinkTime"), buttonDot)
      case modeCountdown:
        if !displayCountdown(display, countdown, buttonDot) {
          mode = modeClock
          sleepTime = DEFAULT_SLEEP
        }
      case modeAlarmError:
        log.Printf("Error: %d\n", error_id)
        display.Print("Err")
      case modeOutput:
        // do nothing
      case modeAlarm:
        // do a strobing 0, light up segments 0 - 5
        if settings.GetBool("strobe") == true {
          display.RefreshOn(false)
          display.SetBlinkRate(sevenseg_backpack.BLINK_OFF)
          display.ClearDisplay()
          for i:=0;i<4;i++ {
            display.SegmentOn(byte(i), byte(alarmSegment), true)
          }
          display.RefreshOn(true)
          alarmSegment = (alarmSegment + 1) % 6
        }
      default:
        log.Printf("Unknown mode: '%d'\n", mode)
    }
  }

  display.DisplayOn(false)
}
