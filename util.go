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
	sleep(time.Duration)
}

type rtc struct {
}

type testClock struct {
	curTime time.Time
}

func (tc testClock) setTime(t time.Time) {
	tc.curTime = t
}

func (tc testClock) now() time.Time {
	return tc.curTime
}

func (tc testClock) sleep(d time.Duration) {
	t := tc.curTime
	// sleep and recheck that the curTime is past the deadline
	// check out the golang clock mocker for ideas on channels to trigger this
	for true {
		if tc.curTime.After(t.Add(d)) {
			return
		}
		time.Sleep(10 * time.Millisecond) // wait for a bit
	}
}

func (tc testClock) add(d time.Duration) {
	tc.curTime.Add(d)
}

func (r rtc) now() time.Time {
	return time.Now()
}

func (r rtc) sleep(d time.Duration) {
	time.Sleep(d)
}

type runtimeConfig struct {
	comms commChannels
	rtc   clock
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
		return buttonMap{}, fmt.Errorf("Could not convert type %T (%v)", rt, rt)
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

func initRuntime(rtc clock) runtimeConfig {
	return runtimeConfig{
		rtc:   rtc,
		comms: initCommChannels()}
}
