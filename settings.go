package main

import (
	"encoding/json"
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

// keep configSettings generic strings, type-convert on the fly
type configSettings struct {
	settings map[string]interface{}
}

func defaultSettings() *configSettings {
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
	s["button_pins"] = []uint8{24, 25}

	on := true
	if runtime.GOARCH == "arm" {
		on = false // default to IRL i2c on the Pi
	}
	s["i2c_simulated"] = on

	return &configSettings{settings: s}
}

func (s *configSettings) settingsFromJSON(data []byte) error {
	tmp := defaultSettings()
	for k, initVal := range tmp.settings {
		var err error
		// we walk through known keys and convert to the target value type
		switch initVal.(type) {
		case uint8:
			var val int64
			valSigned, err := jsonparser.GetInt(data, k)
			if err != nil {
				// try strconv ParseUint
				valString, err2 := jsonparser.GetString(data, k)
				if err2 == nil {
					valSigned, err = strconv.ParseInt(valString, 0, 64)
					val = int64(valSigned)
				}
			} else {
				val = int64(valSigned)
			}
			if val > 255 || val < 0 {
				err = fmt.Errorf("Value %v is out of range for %v", val, k)
			}
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
		case []uint8:
			// unmarshal the string value
			var array []uint8
			err = json.Unmarshal(data, &array)
			if err == nil {
				s.settings[k] = array
			} else {
				log.Println(err)
			}
		default:
			err = fmt.Errorf("Bad type: %T", initVal)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func initSettings() *configSettings {
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

func (s *configSettings) GetString(key string) string {
	switch v := s.settings[key].(type) {
	case string:
		return v
	default:
		log.Fatalf("Could not convert %T to int", v)
		return "noafokinstrang"
	}
}

func (s *configSettings) GetBool(key string) bool {
	switch v := s.settings[key].(type) {
	case bool:
		return v
	default:
		log.Fatalf("Could not convert %T to bool", v)
		return false
	}
}

func (s *configSettings) GetDuration(key string) time.Duration {
	switch v := s.settings[key].(type) {
	case time.Duration:
		return v
	default:
		log.Fatalf("Could not convert %T to time.Duration", v)
		return -1
	}
}

func (s *configSettings) GetByte(key string) byte {
	switch v := s.settings[key].(type) {
	case byte:
		return v
	case int: // cast to byte
		return byte(v)
	default:
		log.Fatalf("Could not convert %T to byte", v)
		return 0xff
	}
}

func (s *configSettings) GetInt(key string) int {
	switch v := s.settings[key].(type) {
	case int:
		return v
	case uint8:
		return int(v)
	default:
		log.Fatalf("Could not convert %T to int", v)
		return -1
	}
}

func (s *configSettings) Dump() {
	for k, v := range s.settings {
		log.Printf("%s : %T: %v\n", k, v, v)
	}
}

/*
func toUInt8(val interface{}) (uint8, error) {
	switch v := val.(type) {
	case uint8:
		return v, nil
	case float64:
		return uint8(v), nil
	case int:
		return uint8(v), nil
	default:
		return 0, errors.New("failed to convert")
	}
}

func toBool(val interface{}) (bool, error) {
	// if it's a string, try strconv.Parse
	switch rt := val.(type) {
	case string:
		return strconv.ParseBool(rt)
	case bool:
		return rt, nil
	case int:
		if rt != 0 {
			return true, nil
		} else {
			return false, nil
		}
	default:
		return false, errors.New("No conversion to bool from ?")
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
				fmt.Println(err)
			}
		}
		return ay, nil
	default:
		return nil, errors.New("No conversion to []uint8 from ")
	}
}

func main() {
	known := make(map[string]interface{})
	known["button_pins"] = []uint8{4, 5}
	known["i2c_bus"] = 0
	known["i2c_device"] = 0x01
	known["blinkTime"] = true

	fmt.Println(data)
	for k, v := range known {
		fmt.Printf("%v : %v\n", k, v)
	}
	var result map[string]interface{}
	err := json.Unmarshal([]byte(data), &result)
	if err != nil {
		fmt.Println(err)
		return
	}

	for k, v := range known {
		fmt.Printf("Result type: %T : %v\n", result[k], result[k])
		switch target := v.(type) {
		case bool:
			known[k], err = toBool(result[k])
			if err != nil {
				fmt.Println(err)
			}
		case []uint8:
			known[k], err = toUInt8Array(result[k])
		case int:
			// straight conversion
			switch rt := result[k].(type) {
			case int:
				known[k] = rt
			default:
				fmt.Printf("No conversion to int from %v: %T\n", k, rt)
			}
		default:
			fmt.Printf("No handler for %v: %T\n", k, target)
		}
	}

	for k, v := range known {
		fmt.Printf("%v : %v\n", k, v)
	}
}*/
