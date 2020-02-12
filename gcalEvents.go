package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
)

func (ge *gcalEvents) fetch(rt runtimeConfig) (*calendar.Events, error) {
	settings := rt.settings
	srv, err := ge.getCalendarService(settings, false)

	if err != nil {
		log.Printf("Failed to get calendar service")
		return nil, err
	}

	// map the calendar to an ID
	calName := settings.GetString(sCalName)
	var id string
	{
		log.Println("get calendar list")
		list, err := srv.CalendarList.List().Do()
		log.Println("process calendar result")
		if err != nil {
			log.Println(err.Error())
			return nil, err
		}
		for _, i := range list.Items {
			if i.Summary == calName {
				id = i.Id
				break
			}
		}
	}

	if id == "" {
		return nil, fmt.Errorf("Could not find calendar %s", calName)
	}
	// get next 10 (?) alarms
	t := rt.clock.Now().Format(time.RFC3339)
	events, err := srv.Events.List(id).
		ShowDeleted(false).
		SingleEvents(true).
		TimeMin(t).
		MaxResults(10).
		OrderBy("startTime").
		Do()
	if err != nil {
		return nil, err
	}

	log.Printf("calendar fetch complete")
	return events, nil
}

func (ge *gcalEvents) getCalendarService(settings configSettings, prompt bool) (*calendar.Service, error) {
	ctx := context.Background()

	b, err := ioutil.ReadFile(settings.GetString(sSecrets) + "/client_secret.json")
	if err != nil {
		log.Printf("Unable to read client secret file: %v", err)
		return nil, err
	}

	// If modifying these scopes, delete your previously saved credentials
	// at ~/.credentials/piclock.json
	config, err := google.ConfigFromJSON(b, calendar.CalendarReadonlyScope)
	if err != nil {
		log.Printf("Unable to parse client secret file to config: %v", err)
		return nil, err
	}
	client := getClient(ctx, config, prompt)

	srv, err := calendar.New(client)
	if err != nil {
		log.Printf("Unable to retrieve calendar Client %v", err)
		return nil, err
	}

	return srv, nil
}

func (ge *gcalEvents) downloadMusicFiles(settings configSettings, cE chan displayEffect) {
	// launch a thread
	go ge.downloadMusicFilesLater(settings, cE)
}

func (ge *gcalEvents) downloadMusicFilesLater(settings configSettings, cE chan displayEffect) {
	// this is currently dumb, it just uses a list from musicDownloads
	// and walks through it, downloading to the music dir
	jsonPath := settings.GetString(sMusicURL)
	log.Printf("Downloading list from " + jsonPath)

	results := make(chan []byte, 20)

	go func() {
		results <- OOBFetch(jsonPath)
	}()

	var files []musicFile
	err := json.Unmarshal(<-results, &files)

	if err != nil {
		log.Printf("Error unmarshalling files: " + err.Error())
		return
	}

	musicPath := settings.GetString(sMusicPath)
	log.Printf("Received a list of %d files", len(files))

	mp3Files := make([]chan []byte, len(files))
	savePaths := make([]string, len(files))

	for i := len(files) - 1; i >= 0; i-- {
		// do we already have that file cached
		savePaths[i] = musicPath + "/" + files[i].Name
		// log.Printf("Checking for " + savePath)
		if _, err := os.Stat(savePaths[i]); os.IsNotExist(err) {
			mp3Files[i] = make(chan []byte, 20)
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

func (ge *gcalEvents) loadAlarms(rt runtimeConfig, loadID int, report bool) {
	// spin up another thread in real life
	go loadAlarmsImpl(rt, loadID, report)
}

func (ge *gcalEvents) generateSecret(rt runtimeConfig) string {
	s1 := rand.NewSource(rt.clock.Now().UnixNano())
	r1 := rand.New(s1)
	return fmt.Sprintf("%04x", r1.Intn(0xFFFF))
}
