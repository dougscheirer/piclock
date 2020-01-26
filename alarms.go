package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"time"
)

type alarm struct {
	ID        string
	Name      string
	When      time.Time
	Effect    int
	Extra     string
	disabled  bool // set to true when we're checking alarms and it fired
	countdown bool // set to true when we're checking alarms and we signaled countdown
}

type loadedPayload struct {
	alarms []alarm
	loadID int
	report bool
}

func toLoadedPayload(val interface{}) (loadedPayload, error) {
	switch v := val.(type) {
	case loadedPayload:
		return v, nil
	default:
		return loadedPayload{}, fmt.Errorf("Bad type: %T", v)
	}
}

const (
	msgLoaded = iota
	msgHandled
	msgReload
	msgMainButton
)

type almStateMsg struct {
	ID  int
	val interface{}
}

type musicFile struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

const (
	almTones = iota
	almMusic
	almRandom
	almFile
	// to pick randomly, provide a max
	almMax
)

func init() {
	// 2 waits, one for runGetAlarams, one for runCheckAlarms
	wg.Add(2)
}

func handledMessage(alm alarm) almStateMsg {
	return almStateMsg{ID: msgHandled, val: alm}
}

func reloadMessage() almStateMsg {
	return almStateMsg{ID: msgReload}
}

func alarmsLoadedMsg(loadID int, alarms []alarm, report bool) almStateMsg {
	return almStateMsg{ID: msgLoaded, val: loadedPayload{loadID: loadID, alarms: alarms, report: report}}
}

func mainButtonAlmMsg(p bool, d time.Duration) almStateMsg {
	return almStateMsg{ID: msgMainButton, val: buttonInfo{pressed: p, duration: d}}
}

func writeAlarms(alarms []alarm, fname string) error {
	output, err := json.Marshal(alarms)
	log.Println(string(output))
	if err != nil {
		return err
	}
	return ioutil.WriteFile(fname, output, 0644)
}

func handledAlarm(alarm alarm, handled map[string]alarm) bool {
	// if the time changed, consider it "unhandled"
	v, ok := handled[alarm.ID]
	if !ok {
		return false
	}
	if v.When != alarm.When {
		return false
	}
	// everything else ignore
	return true
}

func cacheFilename(settings configSettings) string {
	return settings.GetString(sAlarms) + "/alarm.json"
}

func getAlarmsFromCache(rt runtimeConfig) ([]alarm, error) {
	settings := rt.settings
	alarms := make([]alarm, 0)
	if _, err := os.Stat(cacheFilename(settings)); os.IsNotExist(err) {
		return alarms, nil
	}
	data, err := ioutil.ReadFile(cacheFilename(settings))
	if err != nil {
		return alarms, err
	}
	err = json.Unmarshal(data, &alarms)
	if err != nil {
		return alarms, err
	}
	// remove any that are in the "handled" map or the time has passed
	for i := len(alarms) - 1; i >= 0; i-- {
		// TODO: account for countdown time
		if alarms[i].When.Sub(rt.clock.Now()) < 0 {
			// remove is append two slices without the part we don't want
			log.Println(fmt.Sprintf("Discard expired alarm: %s", alarms[i].ID))
			alarms = append(alarms[:i], alarms[i+1:]...)
		}
	}

	return alarms, nil
}

// OOBFetch helper for grabbing http files
func OOBFetch(url string) []byte {
	resp, err := http.Get(url)
	if resp == nil || err != nil || resp.StatusCode != 200 {
		return nil
	}

	body, err2 := ioutil.ReadAll(resp.Body)

	if err2 != nil {
		return nil
	}

	// fmt.Println(string(body))
	return body
}

func runGetAlarms(rt runtimeConfig) {
	defer wg.Done()
	defer func() {
		log.Println("exiting runGetAlarms")
	}()

	settings := rt.settings

	// keep a list of things that we have done
	// TODO: GC the list occassionally
	handledAlarms := map[string]alarm{}
	comms := rt.comms

	var curReloadID int = 0
	var lastRefresh time.Time

	for true {
		// read any messages alarms first
		keepReading := true
		reload := false
		forceReload := false

		if rt.clock.Now().Sub(lastRefresh) > settings.GetDuration(sAlmRefresh) {
			reload = true
		}

		for keepReading {
			select {
			case <-comms.quit:
				log.Println("quit from runGetAlarms")
				return
			case msg := <-comms.getAlarms:
				switch msg.ID {
				case msgHandled:
					alarm, _ := toAlarm(msg.val)
					handledAlarms[alarm.ID] = *alarm
				case msgReload:
					reload = true
					forceReload = true
					comms.effects <- printEffect("rLd", 2*time.Second)
				case msgLoaded:
					// decide if we display a message or not
					// it's possible we launched a bunch of loadAlarms threads
					// and they all eventually unblock. to prevent a bunch of
					// noise, just respond to the one that matches our current ID
					loadedPayload, _ := toLoadedPayload(msg.val)
					if loadedPayload.loadID == curReloadID {
						// force reload -> show alarm count
						if loadedPayload.report {
							comms.effects <- printEffect(fmt.Sprintf("AL:%d", len(loadedPayload.alarms)), 2*time.Second)
						}
					} else {
						log.Printf("Skipping old loadID %v", loadedPayload.loadID)
					}
				default:
					log.Println(fmt.Sprintf("Unknown msg id: %d", msg.ID))
				}
			default:
				keepReading = false
			}
		}

		if reload {
			// launch a thing, it could hang
			loadID := curReloadID + 1
			curReloadID++
			// let the rt decide whether to do it now oe later
			rt.events.loadAlarms(rt, loadID, forceReload)
			lastRefresh = rt.clock.Now()
		} else {
			// wait a little
			rt.clock.Sleep(dAlarmSleep)
		}
	}
}

// return true if they look the same
func compareAlarms(alm1 alarm, alm2 alarm) bool {
	return (alm1.When == alm2.When && alm1.Effect == alm2.Effect &&
		alm1.Name == alm2.Name && alm1.Extra == alm2.Extra)
}

func runCheckAlarm(rt runtimeConfig) {
	defer wg.Done()
	defer func() {
		log.Println("exiting runCheckAlarm")
	}()

	settings := rt.settings
	alarms := make([]alarm, 0)
	comms := rt.comms

	var lastLogSecond = -1

	var lastAlarm *alarm

	for true {
		// try reading from our channel
		select {
		case <-comms.quit:
			log.Println("quit from runCheckAlarm")
			return
		case stateMsg := <-comms.chkAlarms:
			if stateMsg.ID == msgLoaded {
				payload, _ := toLoadedPayload(stateMsg.val)
				alarms = payload.alarms
			} else {
				log.Printf("Ignoring state change message: %v", stateMsg.ID)
			}
		default:
			// continue
		}

		validAlarm := false
		// alarms come in sorted with soonest first
		for index := 0; index < len(alarms); index++ {
			if alarms[index].disabled {
				continue // skip processed alarms
			}

			// if alarms[index] != lastAlarm, run some effects
			if lastAlarm == nil || !compareAlarms(*lastAlarm, alarms[index]) {
				lastAlarm = &alarms[index]
				comms.effects <- printEffect(fmt.Sprintf("AL:%d", index+1), 1*time.Second)
				comms.effects <- printEffect(lastAlarm.When.Format("15:04"), 2*time.Second)
				comms.effects <- printEffect(lastAlarm.When.Format("01.02"), 2*time.Second)
				comms.effects <- printEffect(lastAlarm.When.Format("2006"), 2*time.Second)
			}

			now := rt.clock.Now()
			duration := alarms[index].When.Sub(now)
			if lastLogSecond != now.Second() && now.Second()%30 == 0 {
				lastLogSecond = now.Second()
				log.Println(fmt.Sprintf("Time to next alarm: %ds (%ds to countdown)", duration/time.Second, (duration-settings.GetDuration(sCountdown))/time.Second))
			}

			// light the LED to show we have a pending alarm
			comms.leds <- ledOn(settings.GetInt(sLEDAlm))
			validAlarm = true

			if duration > 0 {
				// start a countdown?
				countdown := settings.GetDuration(sCountdown)
				if duration < countdown && !alarms[index].countdown {
					comms.effects <- setCountdownMode(alarms[0])
					alarms[index].countdown = true
				}
			} else {
				// Set alarm mode
				comms.effects <- setAlarmMode(alarms[index])
				// let getAlarms know we handled it
				comms.getAlarms <- handledMessage(alarms[index])
				alarms[index].disabled = true
			}
			break
		}
		if !validAlarm {
			comms.leds <- ledOff(settings.GetInt(sLEDAlm))
		}
		// take some time off
		rt.clock.Sleep(dAlarmSleep)
	}
}

func loadAlarmsImpl(rt runtimeConfig, loadID int, report bool) {
	comms := rt.comms
	settings := rt.settings

	// also grab all of the music we can
	rt.events.downloadMusicFiles(settings, comms.effects)

	// set error LED now, it should go out almost right away
	comms.leds <- ledMessage(settings.GetInt(sLEDErr), modeBlink75, 0)

	// TODO: handled alarms are not longer considered, need testing
	alarms, err := getAlarmsFromService(rt)
	if err != nil {
		comms.effects <- alarmError(5 * time.Second)
		log.Println(err.Error())
		// try the backup
		alarms, err = getAlarmsFromCache(rt)
		if err != nil {
			// very bad, so...delete and try again later?
			// TODO: more effects
			comms.effects <- alarmError(5 * time.Second)
			log.Printf("Error reading alarm cache: %s\n", err.Error())
			return
		}
		return
	}
	comms.leds <- ledOff(settings.GetInt(sLEDErr))

	msg := alarmsLoadedMsg(loadID, alarms, report)
	// notify state change to runGetAlarms
	comms.getAlarms <- msg
	// notify runCheckAlarms that we have some alarms
	comms.chkAlarms <- msg
}

func getAlarmsFromService(rt runtimeConfig) ([]alarm, error) {
	settings := rt.settings
	events, err := rt.events.fetch(rt)
	var alarms []alarm

	if err != nil {
		return alarms, err
	}

	// remove the cached alarms if they are present
	cacheFile := cacheFilename(settings)
	if _, err := os.Stat(cacheFile); !os.IsNotExist(err) {
		err = os.Remove(cacheFile)
		// an error here is probably a system config issue
		if err != nil {
			// TODO: severe error effect
			log.Printf("Error: %s", err.Error())
			return alarms, err
		}
	}

	// calculate the alarms, write to a file
	if len(events.Items) > 0 {
		for _, i := range events.Items {
			// If the DateTime is an empty string the Event is an all-day Event.
			// So only Date is available.
			if i.Start.DateTime == "" {
				log.Println(fmt.Sprintf("Not a time based alarm, ignoring: %s @ %s", i.Summary, i.Start.Date))
				continue
			}
			var when time.Time
			when, err = time.Parse(time.RFC3339, i.Start.DateTime)
			if err != nil {
				// skip bad formats
				log.Println(err.Error())
				continue
			}

			// TODO: account for countdown time
			if when.Sub(rt.clock.Now()) < 0 {
				log.Println(fmt.Sprintf("Skipping old alarm: %s", i.Id))
				continue
			}

			alm := alarm{ID: i.Id, Name: i.Summary, When: when, disabled: false}

			// look for hashtags (does not work ATM, the gAPI is broken I think)
			log.Printf("Event: %s", i.Summary)
			// priority is arbitrary except for random (default)
			if m, _ := regexp.MatchString("[Mm]usic .*", i.Summary); m {
				alm.Effect = almMusic
				alm.Extra = i.Summary[6:]
			} else if m, _ := regexp.MatchString("[Ff]ile .*", i.Summary); m {
				alm.Effect = almFile
				alm.Extra = i.Summary[5:]
			} else if m, _ := regexp.MatchString("[Tt]one.*", i.Summary); m {
				alm.Effect = almTones // TODO: tone options?
			} else {
				alm.Effect = almRandom
			}

			alarms = append(alarms, alm)
		}

		// cache in a file for later if we go offline
		writeAlarms(alarms, cacheFile)
	}

	return alarms, nil
}
