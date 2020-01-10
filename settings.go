package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/buger/jsonparser"
)

// keep settings generic strings, type-convert on the fly
type settings struct {
	settings map[string]interface{}
}

func defaultSettings() *settings {
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
	s["logFile"] = "/var/log/piclock.log"
	s["cached_alarms"] = false // only use the cache, pretend that gcal is down
	s["musicDownloads"] = "http://192.168.0.105/pimusic"
	s["musicPath"] = "/etc/default/piclock/music"
	s["blinkTime"] = true
	s["strobe"] = true
	s["skiploader"] = false
	s["oauth"] = false

	on := true
	if runtime.GOARCH == "arm" {
		on = false
	}
	s["i2c_simulated"] = on

	return &settings{settings: s}
}

func (s *settings) settingsFromJSON(data []byte) error {
	tmp := defaultSettings()
	for k, initVal := range tmp.settings {
		// ignore missing fields;
		_, err := jsonparser.GetString(data, k)
		if err != nil {
			log.Printf("Skipping key %s", k)
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
				s.settings[k] = byte(val)
			}
		case int:
			s.settings[k], err = jsonparser.GetInt(data, k)
		case int64:
			s.settings[k], err = jsonparser.GetInt(data, k)
		case bool:
			var bVal bool
			bVal, err = jsonparser.GetBoolean(data, k)
			if err != nil {
				// try true and false
				s, _ := jsonparser.GetString(data, k)
				s = strings.ToLower(s)
				switch s {
				case "true":
					bVal = true
				case "false":
					bVal = false
				default:
					// nothing, fail
					return err
				}
			}
			s.settings[k] = bVal
			err = nil
		case time.Duration:
			var dur string
			dur, err = jsonparser.GetString(data, k)
			if err == nil {
				var dur2 time.Duration
				dur2, err = time.ParseDuration(dur)
				if err == nil {
					s.settings[k] = dur2
				}
			}
		case string:
			s.settings[k], err = jsonparser.GetString(data, k)
		default:
			err = fmt.Errorf("Bad type: %T", initVal)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func initSettings() *settings {
	log.Println("initSettings")

	// defaults
	s := defaultSettings()

	// define our flags first
	configFile := flag.String("config", "/etc/default/piclock/piclock.conf", "config file path")
	oauthOnly := flag.Bool("oauth", false, "connect and generate the oauth token")

	// parse the flags
	flag.Parse()

	// oauth?
	if *oauthOnly != false {
		s.settings["oauth"] = true
	}

	// try to open the config file
	data, err := ioutil.ReadFile(*configFile)
	if err != nil {
		log.Fatalf("Could not load conf file '%s', terminating", *configFile)
	}

	log.Println(fmt.Sprintf("Reading configuration from '%s'", *configFile))

	// json parse it
	if err := s.settingsFromJSON(data); err != nil {
		// log a message about crappy JSON?
		log.Fatal(err.Error())
	}

	return s
}

func (s *settings) GetString(key string) string {
	switch v := s.settings[key].(type) {
	case string:
		return v
	default:
		return ""
	}
}

func (s *settings) GetBool(key string) bool {
	switch v := s.settings[key].(type) {
	case bool:
		return v
	default:
		return false
	}
}

func (s *settings) GetDuration(key string) time.Duration {
	switch v := s.settings[key].(type) {
	case time.Duration:
		return v
	default:
		return -1
	}
}

func (s *settings) GetByte(key string) byte {
	switch v := s.settings[key].(type) {
	case byte:
		return v
	case int: // cast to bye
		return byte(v)
	default:
		return 0
	}
}

func (s *settings) GetInt(key string) int {
	switch v := s.settings[key].(type) {
	case int:
		return v
	default:
		return 0
	}
}

func (s *settings) Dump() {
	for k, v := range s.settings {
		log.Printf("%s : %T: %v\n", k, v, v)
	}
}
