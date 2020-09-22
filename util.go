// utility functions
package main

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/jonboulle/clockwork"
	"gopkg.in/natefinch/lumberjack.v2"
)

type commChannels struct {
	quit      chan struct{}
	chkAlarms chan almStateMsg
	effects   chan displayEffect
	getAlarms chan almStateMsg
	leds      chan ledEffect
	configSvc chan configSvcMsg
	ntpVerify chan bool
}

type runtimeConfig struct {
	settings      configSettings
	comms         commChannels
	clock         clockwork.Clock
	sounds        sounds
	buttons       buttons
	display       display
	led           led
	events        events
	configService configService
	logger        flogger
	ntpCheck      ntpcheck
	badTime       bool
}

const dAlarmSleep time.Duration = 100 * time.Millisecond
const dButtonSleep time.Duration = 10 * time.Millisecond
const dEffectSleep time.Duration = 10 * time.Millisecond
const dLEDSleep time.Duration = 10 * time.Millisecond
const dRollingPrint time.Duration = 250 * time.Millisecond
const dPrintDuration time.Duration = 3 * time.Second
const dPrintBriefDuration time.Duration = 1 * time.Second
const dCancelTimeout time.Duration = 5 * time.Second
const dNTPCheckBadSleep time.Duration = 15 * time.Second
const dNTPCheckSleep time.Duration = 5 * time.Minute

const sNextAL string = "next AL..."
const sAt string = "at"
const sNextALIn string = "next AL in..."
const sYorN string = "Y : n"
const sCancel string = "cancel?"
const sPin string = "pin"
const sKey string = "key"
const sPullup string = "pullup"
const sNeedSync string = "need sync..."

func toBool(val interface{}) (bool, error) {
	switch v := val.(type) {
	case bool:
		return v, nil
	case string:
		return strconv.ParseBool(v)
	default:
		return false, fmt.Errorf("Bad type: %T", v)
	}
}

func toInt(val interface{}) (int, error) {
	switch v := val.(type) {
	case int:
		return v, nil
	case string:
		ret, err := strconv.ParseInt(v, 0, 64)
		return int(ret), err
	case float64:
		ret := int(v)
		// make sure it's not really a float
		if v != float64(ret) {
			return -1, fmt.Errorf("Could not convert %T with value %v to int", v, v)
		}
		return ret, nil
	default:
		return -1, fmt.Errorf("Bad type: %T", v)
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
	case string:
		return time.ParseDuration(v)
	default:
		return 0, fmt.Errorf("Bad type: %T", v)
	}
}

func toUInt8(val interface{}) (uint8, error) {
	switch v := val.(type) {
	case uint8:
		return v, nil
	case float64:
		return uint8(v), nil
	case int:
		return uint8(v), nil
	case string:
		ret, err := strconv.ParseInt(v, 0, 8)
		return uint8(ret), err
	default:
		return 0, errors.New("failed to convert")
	}
}

func toUInt8Array(result interface{}) ([]uint8, error) {
	switch rt := result.(type) {
	case []interface{}:
		// convert each value
		var err error
		ay := make([]uint8, len(rt))
		for i := range rt {
			ay[i], err = toUInt8(rt[i])
			if err != nil {
				return ay, err
			}
		}
		return ay, nil
	default:
		return nil, errors.New("No conversion to []uint8 from ")
	}
}

func toButtonMap(result interface{}) (buttonMap, error) {
	switch rt := result.(type) {
	case buttonMap:
		return rt, nil
	case map[string]interface{}:
		// get a pin number and a key (TODO: make these s... enums)
		pin, err := toUInt8(rt[sPin])
		key, err2 := toString(rt[sKey])
		pullup, err3 := toBool(rt[sPullup])

		if err != nil {
			return buttonMap{}, err
		}
		if err2 != nil {
			return buttonMap{}, err2
		}
		if err3 != nil {
			// pick a default
			pullup = true
		}
		return buttonMap{pinNum: pin, key: key, pullup: pullup}, nil
	default:
		return buttonMap{}, fmt.Errorf("Could not convert type %T (%v)", rt, rt)
	}
}

func initCommChannels() commChannels {
	quit := make(chan struct{}, 1)
	alarmChannel := make(chan almStateMsg, 10)
	effectChannel := make(chan displayEffect, 100)
	loaderChannel := make(chan almStateMsg, 10)
	leds := make(chan ledEffect, 100)
	configSvc := make(chan configSvcMsg, 10)
	ntp := make(chan bool, 10)

	return commChannels{
		quit:      quit,
		chkAlarms: alarmChannel,
		effects:   effectChannel,
		getAlarms: loaderChannel,
		leds:      leds,
		configSvc: configSvc,
		ntpVerify: ntp,
	}
}

func initRuntimeConfig(settings configSettings) runtimeConfig {
	var sounds sounds
	var buttons buttons
	var display display
	var led led

	switch settings.GetBool(sDisplay) {
	case true:
		display = &rpioDisplay{}
		led = &rpioLed{}
	default:
		display = &logDisplay{}
		led = &logLed{}
	}

	switch settings.GetString(sButtons) {
	case sKeyboard:
		buttons = &keyButtons{}
	case sRPi:
		buttons = &rpioButtons{}
	default:
		buttons = &noButtons{}
	}

	// do not build audio on platforms
	//       that don't have mplayer (-tags=noaudio)
	switch settings.GetBool(sAudio) {
	case true:
		sounds = &realSounds{}
	default:
		sounds = &noSounds{}
	}

	return runtimeConfig{
		settings:      settings,
		comms:         initCommChannels(),
		clock:         clockwork.NewRealClock(),
		sounds:        sounds,
		buttons:       buttons,
		display:       display,
		led:           led,
		events:        &gcalEvents{},
		configService: &httpConfigService{},
		logger:        &ThreadLogger{name: "main"},
		ntpCheck:      &ntpChecker{},
		badTime:       false,
	}
}

func initTestRuntime(settings configSettings) runtimeConfig {
	// use test modules for sounds/buttons/display/led/events interfaces
	return runtimeConfig{
		settings:      settings,
		comms:         initCommChannels(),
		clock:         clockwork.NewFakeClockAt(time.Date(2020, 01, 26, 0, 0, 0, 0, time.UTC)),
		sounds:        &noSounds{},
		buttons:       &noButtons{},
		display:       &logDisplay{},
		led:           &logLed{},
		events:        &testEvents{},
		configService: &testConfigService{},
		logger:        &ThreadLogger{name: "test"},
		ntpCheck:      &testNtpChecker{},
		badTime:       false,
	}
}

func setupLogging(settings configSettings, append bool) {
	if settings.GetString(sLog) == "" {
		return
	}

	log.SetOutput(&lumberjack.Logger{
		Filename:   settings.GetString(sLog),
		MaxSize:    50, // megabytes
		MaxBackups: 3,
		MaxAge:     28,   //days
		Compress:   true, // disabled by default
	})
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds) // | log.Lshortfile)
}

func calcRolling(s string) time.Duration {
	// the rolling effect pre and post pends 4 spaces, but
	// it really just adds a total of 4 extra cycles
	return time.Duration(len(s)+4) * dRollingPrint
}

// ThreadLogger - special formatting for logs
type ThreadLogger struct {
	name string
}

// Printf - special formatting for logs
func (l *ThreadLogger) Printf(format string, v ...interface{}) {
	log.Printf("%-20s: %s", l.name, fmt.Sprintf(format, v...))
}

// Print - special formatting for logs
func (l *ThreadLogger) Print(v ...interface{}) {
	log.Printf("%-20s: %s", l.name, fmt.Sprint(v...))
}

// Println - special formatting for logs
func (l *ThreadLogger) Println(v ...interface{}) {
	log.Println(fmt.Sprintf("%-20s: %s", l.name, fmt.Sprint(v...)))
}
