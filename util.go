// utility functions
package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/jonboulle/clockwork"
)

type commChannels struct {
	quit      chan struct{}
	chkAlarms chan almStateMsg
	effects   chan displayEffect
	getAlarms chan almStateMsg
	leds      chan ledEffect
	configSvc chan configSvcMsg
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
}

const dAlarmSleep time.Duration = 100 * time.Millisecond
const dButtonSleep time.Duration = 10 * time.Millisecond
const dEffectSleep time.Duration = 10 * time.Millisecond
const dLEDSleep time.Duration = 10 * time.Millisecond
const dRollingPrint time.Duration = 250 * time.Millisecond
const sNextAL string = "next AL in..."
const sYorN string = "Y : n"
const sCancel string = "cancel"
const sPin string = "pin"
const sKey string = "key"
const sPullup string = "pullup"

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

	return commChannels{
		quit:      quit,
		chkAlarms: alarmChannel,
		effects:   effectChannel,
		getAlarms: loaderChannel,
		leds:      leds,
		configSvc: configSvc,
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
	}
}

func setupLogging(settings configSettings, append bool) (*os.File, error) {
	if settings.GetString(sLog) != "" {
		flags := os.O_WRONLY | os.O_CREATE
		if append {
			flags |= os.O_APPEND
		} else {
			flags |= os.O_TRUNC
		}
		f, err := os.OpenFile(settings.GetString(sLog), flags, 0644)
		if err != nil {
			wd, _ := os.Getwd()
			log.Printf("CWD: %s", wd)
			log.Fatal(err)
		}

		// set output of logs to f
		log.SetOutput(f)
		return f, nil
	}

	// default logging is OK
	return nil, nil
}
