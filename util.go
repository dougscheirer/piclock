// utility functions
package main

import (
	"errors"
	"fmt"
	"strconv"
	"time"
)

type commChannels struct {
	quit     chan struct{}
	alarms   chan almStateMsg
	effects  chan displayEffect
	almState chan almStateMsg
	leds     chan ledEffect
}

type clock interface {
	now() time.Time
}

type rtc struct {
}

type wallClock struct {
	curTime time.Time
}

func (r rtc) now() time.Time {
	return time.Now()
}

func (w wallClock) now() time.Time {
	return w.curTime
}

type runtimeConfig struct {
	comms     commChannels
	rtc       clock
	wallClock clock
}

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
		// get a pin number and a key
		pin, err := toUInt8(rt["pin"])
		key, err2 := toString(rt["key"])
		if err != nil {
			return buttonMap{}, err
		}
		if err2 != nil {
			return buttonMap{}, err2
		}
		return buttonMap{pin: pin, key: key}, nil
	default:
		return buttonMap{}, errors.New(fmt.Sprintf("Could not convert type %T (%v)", rt, rt))
	}
}

func initCommChannels() commChannels {
	quit := make(chan struct{}, 1)
	alarmChannel := make(chan almStateMsg, 10)
	effectChannel := make(chan displayEffect, 10)
	loaderChannel := make(chan almStateMsg, 10)
	leds := make(chan ledEffect, 10)

	return commChannels{
		quit:     quit,
		alarms:   alarmChannel,
		effects:  effectChannel,
		almState: loaderChannel,
		leds:     leds}
}

func initRuntime(rtc clock, wallClock clock) runtimeConfig {
	return runtimeConfig{
		rtc:       rtc,
		wallClock: wallClock,
		comms:     initCommChannels()}
}
