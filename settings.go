package main

import (
	"fmt"
	"time"
	"runtime"
	"flag"
	"io/ioutil"
	"github.com/buger/jsonparser"
	"errors"
	"strconv"
)

// keep settings generic strings, type-convert on the fly
type Settings struct {
	settings map[string]interface{};
}

func logMessage(msg string) {
	// TODO: better logging
	_, fname, line, _ := runtime.Caller(1)
	fmt.Printf("%s: %s(%d): %s\n", time.Now().Format(time.UnixDate), fname, line, msg)
}

func defaultSettings() *Settings {
	s := make(map[string]interface{})

	// setting the type here makes the conversion "automatic" later
	s["countdownTime"], _ = time.ParseDuration("1m")
	s["sleepTime"], _ = time.ParseDuration("10ms")
	s["secretPath"] = "/etc/default/piclock"
	s["alarmPath"] = "/etc/default/piclock/alarms"
	s["alarmRefreshTime"], _ = time.ParseDuration("1m")
	s["i2c_bus"] = byte(0)
	s["i2c_device"] = byte(0x70)
	s["calendar"] = "piclock"
	s["debug_dump"] = false
	s["button_simulated"] = ""
	s["cached_alarms"] = false	// only use the cache, pretend that gcal is down

	on := true
	if runtime.GOARCH == "arm" { on = false }
	s["i2c_simulated"] = on

	return &Settings{settings: s}
}

func (this *Settings) settingsFromJSON(data []byte) (error) {
	tmp := defaultSettings()
	for k, initVal := range tmp.settings {
		// ignore missing fields;
		_, err := jsonparser.GetString(data, k)
		if err != nil {
			logMessage(fmt.Sprintf("Skipping key %s",k))
			continue
		}

		switch initVal.(type) {
			case uint8:
				var val uint64
				valSigned, err := jsonparser.GetInt(data, k)
				if err != nil {
					// try strconv ParseUint
					valString, err2 := jsonparser.GetString(data, k)
					if err2 == nil {
						valSigned, err = strconv.ParseInt(valString, 0, 64)
						val = uint64(valSigned)
					}
				} else {
					val = uint64(valSigned)
				}
				// TODO: range check
				if err == nil {
					this.settings[k] = byte(val)
				}
			case int:
				this.settings[k], err = jsonparser.GetInt(data, k)
			case int64:
				this.settings[k], err = jsonparser.GetInt(data, k)
			case bool:
				this.settings[k], err = jsonparser.GetBoolean(data, k)
			case time.Duration:
				var dur string
				dur, err = jsonparser.GetString(data, k)
				if err == nil {
					var dur2 time.Duration
					dur2, err = time.ParseDuration(dur)
					if err == nil {
						this.settings[k] = dur2
					}
				}
			case string:
				this.settings[k], err = jsonparser.GetString(data, k)
			default:
				err = errors.New(fmt.Sprintf("Bad type: %T", initVal))
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func InitSettings() *Settings {
	logMessage("initSettings")

	// defaults
	s := defaultSettings()

	// define our flags first
	configFile := flag.String("config", "/etc/default/piclock/piclock.conf", "config file path")

	// parse the flags
	flag.Parse()

	// try to open it
	data, err := ioutil.ReadFile(*configFile)
	if err != nil {
		logMessage(fmt.Sprintf("Could not load conf file '%s', using defaults", *configFile))
		return s
	}

	logMessage(fmt.Sprintf("Reading configuration from '%s'", *configFile))

	// json parse it
	if err := s.settingsFromJSON(data); err != nil {
		// log a message about crappy JSON?
		logMessage(err.Error())
	}

	return s
}

func (this *Settings) GetString(key string) string {
	switch v := this.settings[key].(type) {
		case string:
			return v
		default:
			return ""
	}
}

func (this *Settings) GetBool(key string) bool {
	switch v := this.settings[key].(type) {
		case bool:
			return v
		default:
			return false
	}
}

func (this *Settings) GetDuration(key string) time.Duration {
	switch v := this.settings[key].(type) {
		case time.Duration:
			return v
		default:
			return -1
	}
}

func (this *Settings) GetByte(key string) byte {
	switch v := this.settings[key].(type) {
		case byte:
			return v
		case int:	// cast to bye
			return byte(v)
		default:
			return 0
	}
}

func (this *Settings) GetInt(key string) int {
	switch v := this.settings[key].(type) {
		case int:
			return v
		default:
			return 0
	}
}

func (this *Settings) Dump() {
	for k, v := range this.settings {
		switch v.(type) {
			case uint8:
				fmt.Printf("%s : %T: %d\n", k, v, v)
			case int:
				fmt.Printf("%s : %T: %d\n", k, v, v)
			case bool:
				fmt.Printf("%s : %T: %t\n", k, v, v)
			case time.Duration:
				fmt.Printf("%s : %T: %d\n", k, v, v)
			case string:
				fmt.Printf("%s : %T: %s\n", k, v, v)
			default:
				fmt.Printf("Bad type: %s: %T\n", k, v)
		}
	}
}