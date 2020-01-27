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

type displayEffect struct {
	id  int
	val interface{}
}

type displayPrint struct {
	s string
	d time.Duration
}

// one of the interface types for displayEffect
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
	eAlarmOn
	eAlarmOff
	eCountdown
)

func init() {
	// runEffects wg
	wg.Add(1)
}

// channel messaging functions
func mainButtonEffect(p bool, d time.Duration) displayEffect {
	return displayEffect{id: eMainButton, val: buttonInfo{pressed: p, duration: d}}
}

func setCountdownMode(alarm alarm) displayEffect {
	return displayEffect{id: eCountdown, val: alarm}
}

func setAlarmMode(alarm alarm) displayEffect {
	return displayEffect{id: eAlarmOn, val: alarm}
}

func cancelAlarmMode(alarm alarm) displayEffect {
	return displayEffect{id: eAlarmOff, val: alarm}
}

func alarmError(d time.Duration) displayEffect {
	return displayEffect{id: eAlarmError, val: d}
}

func toggleDebugDump(on bool) displayEffect {
	return displayEffect{id: eDebug, val: on}
}

func printEffect(s string, d time.Duration) displayEffect {
	return displayEffect{id: ePrint, val: displayPrint{s: s, d: d}}
}

func showLoader(effects chan displayEffect) {
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

func toAlarm(val interface{}) (*alarm, error) {
	switch v := val.(type) {
	case alarm:
		return &v, nil
	default:
		return nil, fmt.Errorf("Bad type: %T", v)
	}
}

func toPrint(val interface{}) (*displayPrint, error) {
	switch v := val.(type) {
	case displayPrint:
		return &v, nil
	default:
		return nil, fmt.Errorf("Bad type: %T", v)
	}
}

func displayClock(rt runtimeConfig, blinkColon bool, dot bool) {
	// standard time display
	colon := "15:04"
	now := rt.clock.Now()
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
	err := rt.display.Print(timeString)
	if err != nil {
		log.Printf("Error: %s\n", err.Error())
	}
}

func displayCountdown(rt runtimeConfig, alarm *alarm, dot bool) bool {
	// calculate 10ths of secs to alarm time
	count := alarm.When.Sub(rt.clock.Now()) / (time.Second / 10)
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
	rt.display.SetBlinkRate(blinkRate)
	rt.display.Print(s)
	return true
}

func playAlarmEffect(rt runtimeConfig, alm *alarm, stop chan bool, done chan bool) {
	musicPath := rt.settings.GetString(sMusicPath)
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
		s1 := rand.NewSource(rt.clock.Now().UnixNano())
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
		rt.sounds.playIt(rt, []string{"250", "340"}, []string{"100ms", "100ms", "100ms", "100ms", "100ms", "2000ms"}, stop, done)
	} else {
		log.Printf("Playing %s", musicFile)
		rt.sounds.playMP3(rt, musicFile, true, stop, done)
	}
}

func stopAlarmEffect(stop chan bool) {
	stop <- true
}

func runEffects(rt runtimeConfig) {
	defer wg.Done()
	defer func() {
		log.Println("exiting runEffects")
	}()

	settings := rt.settings
	comms := rt.comms

	err := rt.display.OpenDisplay(settings)

	if err != nil {
		log.Printf("Error: %s", err.Error())
		return
	}

	// turn on LED dump?
	rt.display.DebugDump(settings.GetBool(sDebug))

	rt.display.SetBrightness(3)
	// ready to rock
	rt.display.DisplayOn(true)

	mode := modeClock
	var countdown *alarm
	var errorID = 0
	alarmSegment := 0
	buttonDot := false

	stopAlarm := make(chan bool, 20)
	done := make(chan bool, 20)

	for true {
		var e displayEffect

		skip := false

		select {
		case <-comms.quit:
			log.Println("quit from runEffects")
			return
		case d := <-done:
			// go back to normal clock mode
			log.Printf("Got a done signal from playEffect: %v", d)
			mode = modeClock
			// TODO: tell checkAlarms that it's over?
		case e = <-comms.effects:
			switch e.id {
			case eDebug:
				v, _ := toBool(e.val)
				rt.display.DebugDump(v)
			case eClock:
				mode = modeClock
			case eCountdown:
				mode = modeCountdown
				countdown, _ = toAlarm(e.val)
			case eAlarmError:
				rt.display.Print("Err")
				d, _ := toDuration(e.val)
				rt.clock.Sleep(d)
			case eTerminate:
				log.Println("terminate")
				return
			case ePrint:
				v, _ := toPrint(e.val)
				log.Printf("Print: %s (%d)", v.s, v.d)
				rt.display.Print(v.s)
				rt.clock.Sleep(v.d)
				skip = true // don't immediately print the clock in clock mode
			case eAlarmOn:
				mode = modeAlarm
				alm, _ := toAlarm(e.val)
				log.Printf(">>>>>>>>>>>>>>> ALARM <<<<<<<<<<<<<<<<<<")
				log.Printf("%s %s %d", alm.Name, alm.When, alm.Effect)
				playAlarmEffect(rt, alm, stopAlarm, done)
			case eAlarmOff:
				mode = modeClock
				alm, _ := toAlarm(e.val)
				log.Printf(">>>>>>>>>>>>>>> STOP ALARM <<<<<<<<<<<<<<<<<<")
				log.Printf("%s %s %d", alm.Name, alm.When, alm.Effect)
				stopAlarmEffect(stopAlarm)
			case eMainButton:
				info, _ := toButtonInfo(e.val)
				buttonDot = info.pressed
			default:
				log.Printf("Unhandled %d\n", e.id)
			}
		default:
			// nothing?
			rt.clock.Sleep(dEffectSleep)
		}

		// skip the mode stuff?
		if skip {
			continue
		}

		switch mode {
		case modeClock:
			displayClock(rt, settings.GetBool(sBlink), buttonDot)
		case modeCountdown:
			if !displayCountdown(rt, countdown, buttonDot) {
				mode = modeClock
			}
		case modeAlarmError:
			log.Printf("Error: %d\n", errorID)
			rt.display.Print("Err")
		case modeOutput:
			// do nothing
		case modeAlarm:
			// do a strobing 0, light up segments 0 - 5
			if settings.GetBool(sStrobe) == true {
				rt.display.RefreshOn(false)
				rt.display.SetBlinkRate(sevenseg_backpack.BLINK_OFF)
				rt.display.ClearDisplay()
				for i := 0; i < 4; i++ {
					rt.display.SegmentOn(byte(i), byte(alarmSegment), true)
				}
				rt.display.RefreshOn(true)
				alarmSegment = (alarmSegment + 1) % 6
			} else {
				rt.display.Print("_-_-")
			}
		default:
			log.Printf("Unknown mode: '%d'\n", mode)
		}
	}

	rt.display.DisplayOn(false)
}
