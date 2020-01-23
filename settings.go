package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"time"
)

// keep configSettings generic strings, type-convert on the fly
type configSettings struct {
	settings map[string]interface{}
}

type buttonMap struct {
	pin uint8
	key string
}

const sCountdown string = "countdownTime"
const sSleep string = "sleepTime"
const sSecrets string = "secretPath"
const sAlarms string = "alarmPath"
const sAlmRefresh string = "alarmRefresh"
const sI2CBus string = "i2cBus"
const sI2CDev string = "i2cDevice"
const sCalName string = "calendar"
const sDebug string = "debugDump"
const sLog string = "logFile"
const sMusicURL string = "musicDownloads"
const sMusicPath string = "musicPath"
const sBlink string = "blinkTime"
const sStrobe string = "strobe"
const sSkipLoader string = "skipLoader"
const sMainBtn string = "mainButton"
const sLEDErr string = "ledErrr"
const sLEDAlm string = "ledAlarm"

func defaultSettings() *configSettings {
	s := make(map[string]interface{})

	// setting the type here makes the conversion "automatic" later
	s[sCountdown], _ = time.ParseDuration("1m")
	s[sSleep], _ = time.ParseDuration("10ms")
	s[sSecrets] = "/etc/default/piclock"
	s[sAlarms] = "/etc/default/piclock/alarms"
	s[sAlmRefresh], _ = time.ParseDuration("1m")
	s[sI2CBus] = byte(0)
	s[sI2CDev] = byte(0x70)
	s[sCalName] = "piclock"
	s[sDebug] = false
	s[sLog] = "/var/log/piclock.log"
	s[sMusicURL] = "http://localhost/pimusic/music.json"
	s[sMusicPath] = "/etc/default/piclock/music"
	s[sBlink] = true
	s[sStrobe] = true
	s[sSkipLoader] = false
	s[sMainBtn] = buttonMap{pin: 25, key: "a"}
	s[sLEDErr] = byte(6)
	s[sLEDAlm] = byte(16)

	return &configSettings{settings: s}
}

func (s *configSettings) settingsFromJSON(data []byte) error {
	tmp := defaultSettings()

	var jsonMap map[string]interface{}
	err := json.Unmarshal([]byte(data), &jsonMap)
	if err != nil {
		return err
	}

	for k, v := range tmp.settings {
		if jsonMap[k] == nil {
			// skip, we will use the default
			continue
		}
		switch target := v.(type) {
		case bool:
			s.settings[k], err = toBool(jsonMap[k])
		case uint8:
			s.settings[k], err = toUInt8(jsonMap[k])
		case []uint8:
			s.settings[k], err = toUInt8Array(jsonMap[k])
		case int:
			s.settings[k], err = toInt(jsonMap[k])
		case string:
			s.settings[k], err = toString(jsonMap[k])
		case time.Duration:
			s.settings[k], err = toDuration(jsonMap[k])
		case buttonMap:
			s.settings[k], err = toButtonMap(jsonMap[k])
		default:
			err = fmt.Errorf("No handler for %v: %T", k, target)
		}

		if err != nil {
			return err
		}
	}

	return nil
}

type cliArgs struct {
	oauth      bool
	configFile string
}

func parseCLIArgs() cliArgs {
	// define our flags first
	configFile := flag.String("config", "/etc/default/piclock/piclock.conf", "config file path")
	oauthOnly := flag.Bool("oauth", false, "connect and generate the oauth token")

	// parse the flags
	flag.Parse()

	args := cliArgs{oauth: false}
	if oauthOnly != nil && *oauthOnly {
		args.oauth = true
	}
	if configFile != nil {
		args.configFile = *configFile
	}

	return args
}

func initSettings(configFile string) *configSettings {
	log.Println("initSettings")

	// defaults
	s := defaultSettings()

	// try to open the config file
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Fatalf("Could not load conf file '%s', terminating", configFile)
	}

	log.Println(fmt.Sprintf("Reading configuration from '%s'", configFile))

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

func (s *configSettings) GetButtonMap(key string) buttonMap {
	switch v := s.settings[key].(type) {
	case buttonMap:
		return v
	default:
		log.Fatalf("Could not convert %T to buttonMap", v)
		return buttonMap{}
	}
}

func (s *configSettings) Dump() {
	for k, v := range s.settings {
		log.Printf("%s : %T: %v\n", k, v, v)
	}
}
