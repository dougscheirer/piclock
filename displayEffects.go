package main

import (
	"container/list"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"dscheirer.com/piclock/sevenseg_backpack"
)

const almDisplay1 string = "_-_-"
const almDisplay2 string = "-_-_"

type displayEffect struct {
	id  int
	val interface{}
}

type displayPrint struct {
	s      string
	d      time.Duration
	cancel chan bool
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
	eLongButton
	eDoubleButton
	eAlarmError
	eTerminate
	ePrint
	ePrintRolling
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

func longButtonEffect(p bool, d time.Duration) displayEffect {
	return displayEffect{id: eLongButton, val: buttonInfo{pressed: p, duration: d}}
}

func doubleButtonEffect(p bool, d time.Duration) displayEffect {
	return displayEffect{id: eDoubleButton, val: buttonInfo{pressed: p, duration: d}}
}

func setCountdownMode(alarm alarm) displayEffect {
	return displayEffect{id: eCountdown, val: alarm}
}

func setAlarmMode(alarm alarm) displayEffect {
	return displayEffect{id: eAlarmOn, val: alarm}
}

func cancelAlarmMode() displayEffect {
	return displayEffect{id: eAlarmOff, val: nil}
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

func printCancelableEffect(s string, d time.Duration, cancel chan bool) displayEffect {
	return displayEffect{id: ePrint, val: displayPrint{s: s, d: d, cancel: cancel}}
}

func printRollingEffect(s string, d time.Duration) displayEffect {
	return displayEffect{id: ePrintRolling, val: displayPrint{s: s, d: d}}
}

func printCancelableRollingEffect(s string, d time.Duration, cancel chan bool) displayEffect {
	return displayEffect{id: ePrintRolling, val: displayPrint{s: s, d: d, cancel: cancel}}
}

func showLoader(effects chan displayEffect) {
	info, err := os.Stat(os.Args[0])
	if err != nil {
		// log error?  non-fatal
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
		// the pi is not great at audio generation
		musicFile = musicPath + "/tones"
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

func printDisplay(rt runtimeConfig, e displayPrint) {
	log.Printf("Print: %s (%d)", e.s, e.d)
	rt.display.Print(e.s)
	// either sleep the entire duration or chunk it out waiting for a cancel
	if e.cancel == nil {
		rt.clock.Sleep(e.d)
	} else {
		start := rt.clock.Now()
		for true {
			select {
			case c := <-e.cancel:
				log.Printf("Got print cancel: %v", c)
				return
			default:
			}
			if rt.clock.Now().Sub(start) > e.d {
				return
			}
			rt.clock.Sleep(dEffectSleep)
		}
	}
}

func printRolling(rt runtimeConfig, e displayPrint) {
	log.Printf("Rolling print: %s (%d)", e.s, e.d)
	// pre/postpend 4/8 spaces, then rotate trhough the string
	// with e.d as the duration on each
	toprint := "    " + e.s + "    "
	for i := 0; i <= len(toprint)-4; i++ {
		_, err := rt.display.PrintOffset(toprint, i)
		if err != nil {
			log.Printf("Error: %s\n", err.Error())
			return
		}
		select {
		case c := <-e.cancel:
			log.Printf("Got rolling print cancel: %v", c)
			return
		default:
		}
		rt.clock.Sleep(e.d)
	}
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

	// stopAlarm will be re-created
	var stopAlarm chan bool = nil
	done := make(chan bool, 20)

	// TODO: put a keepReading around the channel reader?
	printQueue := list.New()

	for true {
		var e displayEffect

		select {
		case <-comms.quit:
			log.Println("quit from runEffects")
			return
		case d := <-done:
			// go back to normal clock mode
			log.Printf("Got a done signal from playEffect: %v", d)
			mode = modeClock
			// tell checkAlarms that it's over?  it could use
			// that information to figure out what to do with
			// button presses
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
			case ePrintRolling:
				v, _ := toPrint(e.val)
				// queue it for later
				printQueue.PushBack(e)
				log.Printf("Queued rolling print: %s (%d)", v.s, v.d)
			case ePrint:
				v, _ := toPrint(e.val)
				// queue it for later
				printQueue.PushBack(e)
				log.Printf("Queued print: %s (%d)", v.s, v.d)
			case eAlarmOn:
				mode = modeAlarm
				alm, _ := toAlarm(e.val)
				log.Printf(">>>>>>>>>>>>>>> ALARM <<<<<<<<<<<<<<<<<<")
				log.Printf("%s %s %d", alm.Name, alm.When, alm.Effect)
				rt.display.SetBlinkRate(sevenseg_backpack.BLINK_OFF)
				// if stopAlarm exists, close it
				if stopAlarm != nil {
					stopAlarmEffect(stopAlarm)
					close(stopAlarm)
				}
				stopAlarm = make(chan bool, 1)
				playAlarmEffect(rt, alm, stopAlarm, done)
			case eAlarmOff:
				mode = modeClock
				// if stopAlarm exists, close it
				if stopAlarm != nil {
					stopAlarmEffect(stopAlarm)
					close(stopAlarm)
					log.Printf(">>>>>>>>>>>>>>> STOP ALARM <<<<<<<<<<<<<<<<<<")
					stopAlarm = nil
				}
				rt.display.SetBlinkRate(sevenseg_backpack.BLINK_OFF)
			case eMainButton:
				info, _ := toButtonInfo(e.val)
				buttonDot = info.pressed
			case eLongButton:
			case eDoubleButton:
			default:
				log.Printf("Unhandled %d\n", e.id)
			}
		default:
			// nothing?
		}

		switch mode {
		case modeClock:
			if printQueue.Len() > 0 {
				log.Printf("Print from queue (%d)", printQueue.Len())
				e := printQueue.Front()
				v := e.Value.(displayEffect)
				switch v.id {
				case ePrint:
					printDisplay(rt, v.val.(displayPrint))
				case ePrintRolling:
					printRolling(rt, v.val.(displayPrint))
				}
				printQueue.Remove(e)
			} else {
				displayClock(rt, settings.GetBool(sBlink), buttonDot)
			}
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
				rt.display.ClearDisplay()
				for i := 0; i < 4; i++ {
					rt.display.SegmentOn(byte(i), byte(alarmSegment), true)
				}
				rt.display.RefreshOn(true)
				alarmSegment = (alarmSegment + 1) % 6
			} else {
				if (rt.clock.Now().Second())%2 == 0 {
					rt.display.Print(almDisplay1)
				} else {
					rt.display.Print(almDisplay2)
				}
			}
		default:
			log.Printf("Unknown mode: '%d'\n", mode)
		}

		rt.clock.Sleep(dEffectSleep)
	}

	rt.display.DisplayOn(false)
}
