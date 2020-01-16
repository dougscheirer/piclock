package main

import (
	"encoding/json"
	"errors"
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

type loaderMsg struct {
	msg   string
	alarm alarm
	val   interface{}
}

type checkMsg struct {
	displayCurrent bool
	alarms         []alarm
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

func handledMessage(alm alarm) loaderMsg {
	return loaderMsg{msg: "handled", alarm: alm}
}

func reloadMessage() loaderMsg {
	return loaderMsg{msg: "reload"}
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

func cacheFilename(settings *settings) string {
	return settings.GetString("alarmPath") + "/alarm.json"
}

func getAlarmsFromService(settings *settings, runtime runtimeConfig, handled map[string]alarm) ([]alarm, error) {
	alarms := make([]alarm, 0)
	srv := getCalenderService(settings, false)

	// TODO: if it wasn't available, send an alarm message
	if srv == nil {
		return alarms, errors.New("Failed to get calendar service")
	}

	// map the calendar to an ID
	calName := settings.GetString("calendar")
	var id string
	{
		log.Println("get calendar list")
		list, err := srv.CalendarList.List().Do()
		if err != nil {
			log.Println(err.Error())
			return alarms, err
		}
		for _, i := range list.Items {
			if i.Summary == calName {
				id = i.Id
				break
			}
		}
	}

	if id == "" {
		return alarms, fmt.Errorf("Could not find calendar %s", calName)
	}
	// get next 10 (?) alarms
	t := runtime.wallClock.now().Format(time.RFC3339)
	events, err := srv.Events.List(id).
		ShowDeleted(false).
		SingleEvents(true).
		TimeMin(t).
		MaxResults(10).
		OrderBy("startTime").
		Do()
	if err != nil {
		return alarms, err
	}

	// remove the cached alarms if they are present
	cacheFile := cacheFilename(settings)
	if _, err := os.Stat(cacheFile); !os.IsNotExist(err) {
		err = os.Remove(cacheFile)
		// an error here is a system config issue
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
				log.Println(fmt.Sprintf("Not a time based alarm: %s @ %s", i.Summary, i.Start.Date))
				continue
			}
			var when time.Time
			when, err = time.Parse(time.RFC3339, i.Start.DateTime)
			if err != nil {
				// skip bad formats
				log.Println(err.Error())
				continue
			}

			if when.Sub(runtime.wallClock.now()) < 0 {
				log.Println(fmt.Sprintf("Skipping old alarm: %s", i.Id))
				continue
			}

			alm := alarm{ID: i.Id, Name: i.Summary, When: when, disabled: false}

			// look for hashtags (does not work ATM, the gAPI is broken I think)
			log.Println(i.Summary)
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

			// has this one been handled?
			if handledAlarm(alm, handled) {
				log.Println(fmt.Sprintf("Skipping handled alarm: %s", alm.ID))
				continue
			}

			alarms = append(alarms, alm)
		}

		// cache in a file for later if we go offline
		writeAlarms(alarms, cacheFile)
	}

	return alarms, nil
}

func getAlarmsFromCache(settings *settings, runtime runtimeConfig, handled map[string]alarm) ([]alarm, error) {
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
		if handledAlarm(alarms[i], handled) {
			// remove is append two slices without the part we don't want
			log.Println(fmt.Sprintf("Discard handled alarm: %s", alarms[i].ID))
			alarms = append(alarms[:i], alarms[i+1:]...)
		}
		if alarms[i].When.Sub(runtime.wallClock.now()) < 0 {
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

func downloadMusicFiles(settings *settings, cE chan effect) {
	// this is currently dumb, it just uses a list from musicDownloads
	// and walks through it, downloading to the music dir
	jsonPath := settings.GetString("musicDownloads")
	log.Printf("Downloading list from " + jsonPath)

	results := make(chan []byte, 1)

	go func() {
		results <- OOBFetch(jsonPath)
	}()

	var files []musicFile
	err := json.Unmarshal(<-results, &files)

	if err != nil {
		log.Printf("Error unmarshalling files: " + err.Error())
		return
	}

	musicPath := settings.GetString("musicPath")
	log.Printf("Received a list of %d files", len(files))

	mp3Files := make([]chan []byte, len(files))
	savePaths := make([]string, len(files))

	for i := len(files) - 1; i >= 0; i-- {
		// do we already have that file cached
		savePaths[i] = musicPath + "/" + files[i].Name
		// log.Printf("Checking for " + savePath)
		if _, err := os.Stat(savePaths[i]); os.IsNotExist(err) {
			mp3Files[i] = make(chan []byte, 1)
			go func(i int) {
				log.Println(fmt.Sprintf("Downloading %s [%s]", files[i].Name, files[i].Path))
				mp3Files[i] <- OOBFetch(files[i].Path)
			}(i)
		}
	}

	for i := len(files) - 1; i >= 0; i-- {
		if mp3Files[i] == nil {
			continue
		}

		// write the file
		data := <-mp3Files[i]
		if data == nil || len(data) == 0 {
			log.Printf("Skipping nil data for %s", savePaths[i])
			continue
		}

		log.Printf("Saving %s", savePaths[i])
		err = ioutil.WriteFile(savePaths[i], data, 0644)
		if err != nil {
			// handle error
			log.Println(fmt.Sprintf("Failed to write %s: %s", savePaths[i], err.Error()))
			continue
		}
	}
}

func runGetAlarms(settings *settings, runtime runtimeConfig) {
	defer wg.Done()
	defer func() {
		log.Println("exiting runGetAlarms")
	}()

	// keep a list of things that we have done
	// TODO: GC the list occassionally
	handledAlarms := map[string]alarm{}
	comms := runtime.comms

	var lastRefresh time.Time

	for true {
		// read any messages alarms first
		keepReading := true
		reload := false
		displayCurrent := false
		forceReload := false

		if runtime.rtc.now().Sub(lastRefresh) > settings.GetDuration("alarmRefreshTime") {
			reload = true
		}

		for keepReading {
			select {
			case <-comms.quit:
				log.Println("quit from runGetAlarms")
				return
			case msg := <-comms.loader:
				switch msg.msg {
				case "handled":
					handledAlarms[msg.alarm.ID] = msg.alarm
					// reload sends a new list without the ones that are handled
					displayCurrent = true
				case "reload":
					displayCurrent = true
					reload = true
					forceReload = true
					comms.effects <- printEffect("rLd", 2*time.Second)
				default:
					log.Println(fmt.Sprintf("Unknown msg id: %s", msg.msg))
				}
			default:
				keepReading = false
			}
		}

		if reload {
			// launch a thread to grab all of the music we can
			go downloadMusicFiles(settings, comms.effects)

			errorLED(false)

			alarms, err := getAlarmsFromService(settings, runtime, handledAlarms)
			if err != nil {
				errorLED(true)
				comms.effects <- alarmError(5 * time.Second)
				log.Println(err.Error())
				// try the backup
				alarms, err = getAlarmsFromCache(settings, runtime, handledAlarms)
				if err != nil {
					// very bad, so...delete and try again later?
					// TODO: more effects
					comms.effects <- alarmError(5 * time.Second)
					log.Printf("Error reading alarm cache: %s\n", err.Error())
					time.Sleep(time.Second)
					continue
				}
			}

			// force reload -> show alarm count
			// normal reload -> only show if > 0
			if forceReload || len(alarms) > 0 {
				comms.effects <- printEffect(fmt.Sprintf("AL:%d", len(alarms)), 2*time.Second)
			}

			lastRefresh = runtime.rtc.now()

			// tell cA that we have some alarms?
			comms.alarms <- checkMsg{alarms: alarms, displayCurrent: displayCurrent}
		} else {
			// wait a little
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func runCheckAlarm(settings *settings, runtime runtimeConfig) {
	defer wg.Done()
	defer func() {
		log.Println("exiting runCheckAlarms")
	}()

	alarms := make([]alarm, 0)
	comms := runtime.comms

	var lastLogSecond = -1

	var lastAlarm *alarm

	for true {
		// try reading from our channel
		select {
		case <-comms.quit:
			log.Println("quit from runCheckAlarm")
			return
		case checkMsg := <-comms.alarms:
			alarms = checkMsg.alarms
			if checkMsg.displayCurrent {
				lastAlarm = nil
			}
		default:
			// continue
		}

		// alarms come in sorted with soonest first
		for index := 0; index < len(alarms); index++ {
			if alarms[index].disabled {
				continue // skip processed alarms
			}

			// if alarms[index] != lastAlarm, run some effects
			if lastAlarm == nil || lastAlarm.When != alarms[index].When {
				lastAlarm = &alarms[index]
				comms.effects <- printEffect(fmt.Sprintf("AL:%d", index+1), 1*time.Second)
				comms.effects <- printEffect(lastAlarm.When.Format("15:04"), 2*time.Second)
				comms.effects <- printEffect(lastAlarm.When.Format("01.02"), 2*time.Second)
				comms.effects <- printEffect(lastAlarm.When.Format("2006"), 2*time.Second)
			}

			now := runtime.wallClock.now()
			duration := alarms[index].When.Sub(now)
			if lastLogSecond != now.Second() && now.Second()%30 == 0 {
				lastLogSecond = now.Second()
				log.Println(fmt.Sprintf("Time to next alarm: %ds (%ds to countdown)", duration/time.Second, (duration-settings.GetDuration("countdownTime"))/time.Second))
			}

			if duration > 0 {
				// start a countdown?
				countdown := settings.GetDuration("countdownTime")
				if duration < countdown && !alarms[index].countdown {
					comms.effects <- setCountdownMode(alarms[0])
					alarms[index].countdown = true
				}
			} else {
				// Set alarm mode
				comms.effects <- setAlarmMode(alarms[index])
				// let someone know we handled it
				comms.loader <- handledMessage(alarms[index])
				alarms[index].disabled = true
			}
			break
		}
		// take some time off
		time.Sleep(100 * time.Millisecond)
	}
}
