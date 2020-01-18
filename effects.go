package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"piclock/sevenseg_backpack"
	"time"
)

type effect struct {
	id  int
	val interface{}
}

type print struct {
	s string
	d time.Duration
}

type buttonInfo struct {
	pressed  bool
	duration time.Duration
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

func init() {
	// runEffects wg
	wg.Add(1)
}

// channel messaging functions
func mainButton(p bool, d time.Duration) effect {
	return effect{id: eMainButton, val: buttonInfo{pressed: p, duration: d}}
}

func setCountdownMode(alarm alarm) effect {
	return effect{id: eCountdown, val: alarm}
}

func setAlarmMode(alarm alarm) effect {
	return effect{id: eAlarm, val: alarm}
}

func alarmError(d time.Duration) effect {
	return effect{id: eAlarmError, val: d}
}

func toggleDebugDump(on bool) effect {
	return effect{id: eDebug, val: on}
}

func printEffect(s string, d time.Duration) effect {
	return effect{id: ePrint, val: print{s: s, d: d}}
}

func showLoader(effects chan effect) {
	info, err := os.Stat(os.Args[0])
	if err != nil {
		// TODO: log error?  non-fatal
		log.Printf("%v", err)
		return
	}

	effects <- printEffect("bLd.", 1500*time.Millisecond)
	effects <- printEffect("----", 500*time.Millisecond)
	effects <- printEffect(info.ModTime().Format("15:04"), 1500*time.Millisecond)
	effects <- printEffect("----", 500*time.Millisecond)
	effects <- printEffect(info.ModTime().Format("01.02"), 1500*time.Millisecond)
	effects <- printEffect("----", 500*time.Millisecond)
	effects <- printEffect(info.ModTime().Format("2006"), 1500*time.Millisecond)
	effects <- printEffect("----", 500*time.Millisecond)
}

func replaceAtIndex(in string, r rune, i int) string {
	out := []rune(in)
	out[i] = r
	return string(out)
}

func toButtonInfo(val interface{}) (*buttonInfo, error) {
	switch v := val.(type) {
	case buttonInfo:
		return &v, nil
	default:
		return nil, fmt.Errorf("Bad type: %T", v)
	}
}

func toBool(val interface{}) (bool, error) {
	switch v := val.(type) {
	case bool:
		return v, nil
	default:
		return false, fmt.Errorf("Bad type: %T", v)
	}
}

func toInt(val interface{}) (int, error) {
	switch v := val.(type) {
	case int:
		return v, nil
	default:
		return -1, fmt.Errorf("Bad type: %T", v)
	}
}

func toAlarm(val interface{}) (*alarm, error) {
	switch v := val.(type) {
	case alarm:
		return &v, nil
	default:
		return nil, fmt.Errorf("Bad type: %T", v)
	}
}

func toString(val interface{}) (string, error) {
	switch v := val.(type) {
	case string:
		return v, nil
	default:
		return "", fmt.Errorf("Bad type: %T", v)
	}
}

func toDuration(val interface{}) (time.Duration, error) {
	switch v := val.(type) {
	case time.Duration:
		return v, nil
	default:
		return 0, fmt.Errorf("Bad type: %T", v)
	}
}

func toPrint(val interface{}) (*print, error) {
	switch v := val.(type) {
	case print:
		return &v, nil
	default:
		return nil, fmt.Errorf("Bad type: %T", v)
	}
}

func displayClock(runtime runtimeConfig, display *sevenseg_backpack.Sevenseg, blinkColon bool, dot bool) {
	// standard time display
	colon := "15:04"
	now := runtime.wallClock.now()
	if blinkColon && now.Second()%2 == 0 {
		// no space required for the colon
		colon = "1504"
	}

	timeString := now.Format(colon)
	if timeString[0] == '0' {
		timeString = replaceAtIndex(timeString, ' ', 0)
	}
	if dot {
		timeString += "."
	}
	err := display.Print(timeString)
	if err != nil {
		log.Printf("Error: %s\n", err.Error())
	}
}

func displayCountdown(runtime runtimeConfig, display *sevenseg_backpack.Sevenseg, alarm *alarm, dot bool) bool {
	// calculate 10ths of secs to alarm time
	count := alarm.When.Sub(runtime.wallClock.now()) / (time.Second / 10)
	if count > 9999 {
		count = 9999
	} else if count <= 0 {
		return false
	}
	s := fmt.Sprintf("%d.%d", count/10, count%10)
	if dot {
		s += "."
	}
	var blinkRate uint8 = sevenseg_backpack.BLINK_OFF
	if count < 100 {
		blinkRate = sevenseg_backpack.BLINK_2HZ
	}
	display.SetBlinkRate(blinkRate)
	display.Print(s)
	return true
}

func playAlarmEffect(settings *settings, alm *alarm, stop chan bool, runtime runtimeConfig) {
	musicPath := settings.GetString("musicPath")
	var musicFile string
	playTones := false

	switch alm.Effect {
	case almMusic:
		musicFile = musicPath + "/" + alm.Extra
	case almFile:
		musicFile = musicPath + "/" + alm.Extra
	case almTones:
		playTones = true
		return
	default:
		// play a random mp3 in the cache
		s1 := rand.NewSource(runtime.rtc.now().UnixNano())
		r1 := rand.New(s1)

		files, err := filepath.Glob(musicPath + "/*")
		if err != nil {
			log.Println(err)
			break
		}
		if len(files) > 0 {
			musicFile = files[r1.Intn(len(files))]
		} else {
			playTones = true
		}
	}

	if !playTones {
		// make sure the file exists
		fstat, err := os.Stat(musicFile)
		if err != nil || fstat == nil {
			playTones = true
		}
	}

	if playTones {
		log.Printf("Playing tones")
		playIt([]string{"250", "340"}, []string{"100ms", "100ms", "100ms", "100ms", "100ms", "2000ms"}, stop)
	} else {
		log.Printf("Playing %s", musicFile)
		playMP3(musicFile, true, stop)
	}
}

func stopAlarmEffect(stop chan bool) {
	stop <- true
}

func runEffects(settings *settings, runtime runtimeConfig) {
	defer wg.Done()
	defer func() {
		log.Println("exiting runEffects")
	}()

	comms := runtime.comms

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
	var countdown *alarm
	var errorID = 0
	alarmSegment := 0
	defaultSleep := settings.GetDuration("sleepTime")
	sleepTime := defaultSleep
	buttonPressActed := false
	buttonDot := false

	stopAlarm := make(chan bool, 1)

	for true {
		var e effect

		skip := false

		select {
		case <-comms.quit:
			log.Println("quit from runEffects")
			return
		case e = <-comms.effects:
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
				sleepTime = 10 * time.Millisecond
				log.Printf(">>>>>>>>>>>>>>> ALARM <<<<<<<<<<<<<<<<<<")
				log.Printf("%s %s %d", alm.Name, alm.When, alm.Effect)
				go playAlarmEffect(settings, alm, stopAlarm, runtime)
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
							sleepTime = defaultSleep
							buttonPressActed = true
							display.SetBlinkRate(0)
							stopAlarmEffect(stopAlarm)
						case modeCountdown:
							// cancel the alarm
							mode = modeClock
							comms.loader <- handledMessage(*countdown)
							countdown = nil
							buttonPressActed = true
						case modeClock:
							// more than 5 seconds is "reload"
							if info.duration > 4*time.Second {
								comms.loader <- reloadMessage()
								buttonPressActed = true
							}
						default:
							log.Printf("No action for mode %d", mode)
						}
					}
				} else {
					buttonPressActed = false
					log.Printf("Main button released: %dms", info.duration/time.Millisecond)
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
			displayClock(runtime, display, settings.GetBool("blinkTime"), buttonDot)
		case modeCountdown:
			if !displayCountdown(runtime, display, countdown, buttonDot) {
				mode = modeClock
				sleepTime = defaultSleep
			}
		case modeAlarmError:
			log.Printf("Error: %d\n", errorID)
			display.Print("Err")
		case modeOutput:
			// do nothing
		case modeAlarm:
			// do a strobing 0, light up segments 0 - 5
			if settings.GetBool("strobe") == true {
				display.RefreshOn(false)
				display.SetBlinkRate(sevenseg_backpack.BLINK_OFF)
				display.ClearDisplay()
				for i := 0; i < 4; i++ {
					display.SegmentOn(byte(i), byte(alarmSegment), true)
				}
				display.RefreshOn(true)
				alarmSegment = (alarmSegment + 1) % 6
			} else {
				display.Print("_-_-")
			}
		default:
			log.Printf("Unknown mode: '%d'\n", mode)
		}
	}

	display.DisplayOn(false)
}
