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

type Alarm struct {
	Id        string
	Name      string
	When      time.Time
	Effect    int
	Extra     string
	disabled  bool // set to true when we're checking alarms and it fired
	countdown bool // set to true when we're checking alarms and we signaled countdown
}

type LoaderMsg struct {
	msg   string
	alarm Alarm
	val   interface{}
}

type CheckMsg struct {
	displayCurrent bool
	alarms         []Alarm
}

type MusicFile struct {
	Name string
	Path string
}

const (
	almTones = iota
	almMusic
	almRandom
	almFile
	// to pick randomly, provide a max
	almMax
)

func handledMessage(alm Alarm) LoaderMsg {
	return LoaderMsg{msg: "handled", alarm: alm}
}

func reloadMessage() LoaderMsg {
	return LoaderMsg{msg: "reload"}
}

func writeAlarms(alarms []Alarm, fname string) error {
	output, err := json.Marshal(alarms)
	log.Println(string(output))
	if err != nil {
		return err
	}
	return ioutil.WriteFile(fname, output, 0644)
}

func handledAlarm(alarm Alarm, handled map[string]Alarm) bool {
	// if the time changed, consider it "unhandled"
	v, ok := handled[alarm.Id]
	if !ok {
		return false
	}
	if v.When != alarm.When {
		return false
	}
	// everything else ignore
	return true
}

func cacheFilename(settings *Settings) string {
	return settings.GetString("alarmPath") + "/alarm.json"
}

func getAlarmsFromService(settings *Settings, handled map[string]Alarm) ([]Alarm, error) {
	alarms := make([]Alarm, 0)
	srv := GetCalenderService(settings, false)

	// TODO: if it wasn't available, send an Alarm message
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
		return alarms, errors.New(fmt.Sprintf("Could not find calendar %s", calName))
	}
	// get next 10 (?) alarms
	t := time.Now().Format(time.RFC3339)
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

			if when.Sub(time.Now()) < 0 {
				log.Println(fmt.Sprintf("Skipping old alarm: %s", i.Id))
				continue
			}

			alm := Alarm{Id: i.Id, Name: i.Summary, When: when, disabled: false}

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
				log.Println(fmt.Sprintf("Skipping handled alarm: %s", alm.Id))
				continue
			}

			alarms = append(alarms, alm)
		}

		// cache in a file for later if we go offline
		writeAlarms(alarms, cacheFile)
	}

	// if we're developing, make an alarm 1 minute in the future
	if settings.GetBool("fake_alarm") {
		alm := Alarm{Id: "thisistotallyfake", Name: "who cares", When: time.Now().Add(time.Duration(1) * time.Minute), disabled: false, Effect: almRandom}
		alarms = append(alarms, alm)
		writeAlarms(alarms, cacheFile)
	}

	return alarms, nil
}

func getAlarmsFromCache(settings *Settings, handled map[string]Alarm) ([]Alarm, error) {
	alarms := make([]Alarm, 0)
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
			log.Println(fmt.Sprintf("Discard handled alarm: %s", alarms[i].Id))
			alarms = append(alarms[:i], alarms[i+1:]...)
		}
		if alarms[i].When.Sub(time.Now()) < 0 {
			// remove is append two slices without the part we don't want
			log.Println(fmt.Sprintf("Discard expired alarm: %s", alarms[i].Id))
			alarms = append(alarms[:i], alarms[i+1:]...)
		}
	}

	return alarms, nil
}

func downloadMusicFiles(settings *Settings, cE chan Effect) {
	// this is currently dumb, it just uses a list from musicDownloads (music.json)
	// and walks through it, downloading to the music dir
	jsonPath := settings.GetString("musicDownloads") + "/music.json"
	log.Printf("Downloading list from " + jsonPath)
	resp, err := http.Get(jsonPath)
	if err != nil {
		log.Println("Error fetching music.json: " + err.Error())
		// cE <- printEffect("Err", 2*time.Second )
		return
	}
	if resp.StatusCode != 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		log.Println("Error fetching music.json: " + string(body))
		// cE <- printEffect("Err", 2*time.Second )
		return
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	files := make([]MusicFile, 0)
	err = json.Unmarshal(body, &files)
	if err != nil {
		log.Printf("Error unmarshalling files: " + err.Error())
		return
	}
	musicPath := settings.GetString("musicPath")
	log.Printf("Received a list of %d files", len(files))
	for i := len(files) - 1; i >= 0; i-- {
		// do we already have that file cached
		savePath := musicPath + "/" + files[i].Name
		// log.Printf("Checking for " + savePath)
		if _, err := os.Stat(savePath); os.IsNotExist(err) {
			// download it
			log.Println(fmt.Sprintf("Downloading %s [%s]", files[i].Name, files[i].Path))
			resp, err := http.Get(files[i].Path)
			if err != nil {
				// handle error
				log.Println(fmt.Sprintf("Failed to download %s: %s", files[i].Name, err.Error()))
				continue
			}
			if resp.StatusCode != 200 {
				log.Println(fmt.Sprintf("Received bad status code: %d", resp.StatusCode))
				continue
			}
			defer resp.Body.Close()
			body, err := ioutil.ReadAll(resp.Body)
			// write the file
			err = ioutil.WriteFile(savePath, body, 0644)
			if err != nil {
				// handle error
				log.Println(fmt.Sprintf("Failed to write %s: %s", savePath, err.Error()))
				continue
			}
		}
	}
}

func runGetAlarms(settings *Settings, comms CommChannels) {
	defer wg.Done()

	// keep a list of things that we have done
	// TODO: GC the list occassionally
	handledAlarms := map[string]Alarm{}

	var lastRefresh time.Time

	for true {
		// read any messages alarms first
		keepReading := true
		reload := false
		displayCurrent := false

		if time.Now().Sub(lastRefresh) > settings.GetDuration("alarmRefreshTime") {
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
					handledAlarms[msg.alarm.Id] = msg.alarm
					// reload sends a new list without the ones that are handled
					displayCurrent = true
				case "reload":
					displayCurrent = true
					reload = true
					comms.effects <- printEffect("rLd", 2*time.Second)
				default:
					log.Println(fmt.Sprintf("Unknown msg id: %s", msg.msg))
				}
			default:
				keepReading = false
			}
		}

		if reload {
			alarms, err := getAlarmsFromService(settings, handledAlarms)
			if err != nil {
				comms.effects <- alarmError(5 * time.Second)
				log.Println(err.Error())
				// try the backup
				alarms, err = getAlarmsFromCache(settings, handledAlarms)
				if err != nil {
					// very bad, so...delete and try again later?
					// TODO: more effects
					comms.effects <- alarmError(5 * time.Second)
					log.Printf("Error reading alarm cache: %s\n", err.Error())
					time.Sleep(time.Second)
					continue
				}
			}

			// launch a thread to grab all of the music we can
			go downloadMusicFiles(settings, comms.effects)

			lastRefresh = time.Now()

			// tell cA that we have some alarms?
			comms.alarms <- CheckMsg{alarms: alarms, displayCurrent: displayCurrent}
		} else {
			// wait a little
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func runCheckAlarm(settings *Settings, comms CommChannels) {
	defer wg.Done()

	alarms := make([]Alarm, 0)
	var lastLogSecond = -1

	var lastAlarm *Alarm

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
				comms.effects <- printEffect("AL:", 2*time.Second)
				comms.effects <- printEffect(lastAlarm.When.Format("15:04"), 3*time.Second)
				comms.effects <- printEffect(lastAlarm.When.Format("01.02"), 3*time.Second)
				comms.effects <- printEffect(lastAlarm.When.Format("2006"), 3*time.Second)
			}

			now := time.Now()
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
