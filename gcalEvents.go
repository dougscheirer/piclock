package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
)

func (ge *gcalEvents) fetch(rt runtimeConfig) (*calendar.Events, error) {
	settings := rt.settings
	srv, err := ge.getCalendarService(rt, false)

	if err != nil {
		rt.logger.Printf("Failed to get calendar service")
		return nil, err
	}

	// map the calendar to an ID
	calName := settings.GetString(sCalName)
	var id string
	{
		rt.logger.Println("get calendar list")
		list, err := srv.CalendarList.List().Do()
		rt.logger.Println("process calendar result")
		if err != nil {
			rt.logger.Println(err.Error())
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

	rt.logger.Printf("calendar fetch complete")
	return events, nil
}

func (ge *gcalEvents) getCalendarService(rt runtimeConfig, prompt bool) (*calendar.Service, error) {
	ctx := context.Background()

	b, err := ioutil.ReadFile(rt.settings.GetString(sSecrets) + "/client_secret.json")
	if err != nil {
		rt.logger.Printf("Unable to read client secret file: %v", err)
		return nil, err
	}

	// If modifying these scopes, delete your previously saved credentials
	// at ~/.credentials/piclock.json
	config, err := google.ConfigFromJSON(b, calendar.CalendarReadonlyScope)
	if err != nil {
		rt.logger.Printf("Unable to parse client secret file to config: %v", err)
		return nil, err
	}
	client, err := getClient(ctx, config, prompt)
	if err != nil {
		rt.logger.Printf("Unable to retrieve calendar Client %v", err)
		return nil, err
	}

	srv, err2 := calendar.New(client)
	if err2 != nil {
		rt.logger.Printf("Unable to retrieve calendar Client %v", err2)
		return nil, err2
	}

	return srv, nil
}

func (ge *gcalEvents) downloadMusicFiles(rt runtimeConfig, cE chan displayEffect) {
	// launch a thread
	go ge.downloadMusicFilesLater(rt, cE)
}

func (ge *gcalEvents) downloadMusicFilesLater(rt runtimeConfig, cE chan displayEffect) {
	// this is currently dumb, it just uses a list from musicDownloads
	// and walks through it, downloading to the music dir
	jsonPath := rt.settings.GetString(sMusicURL)
	rt.logger.Printf("Downloading list from " + jsonPath)

	results := make(chan []byte, 20)

	go func() {
		results <- OOBFetch(jsonPath)
	}()

	var files []musicFile
	err := json.Unmarshal(<-results, &files)

	if err != nil {
		rt.logger.Printf("Error unmarshalling files: " + err.Error())
		return
	}

	musicPath := rt.settings.GetString(sMusicPath)
	rt.logger.Printf("Received a list of %d files", len(files))

	mp3Files := make([]chan []byte, len(files))
	savePaths := make([]string, len(files))

	for i := len(files) - 1; i >= 0; i-- {
		// do we already have that file cached
		savePaths[i] = musicPath + "/" + files[i].Name
		// rt.logger.Printf("Checking for " + savePath)
		if _, err := os.Stat(savePaths[i]); os.IsNotExist(err) {
			mp3Files[i] = make(chan []byte, 20)
			go func(i int) {
				rt.logger.Println(fmt.Sprintf("Downloading %s [%s]", files[i].Name, files[i].Path))
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
			rt.logger.Printf("Skipping nil data for %s", savePaths[i])
			continue
		}

		rt.logger.Printf("Saving %s", savePaths[i])
		err = ioutil.WriteFile(savePaths[i], data, 0644)
		if err != nil {
			// handle error
			rt.logger.Println(fmt.Sprintf("Failed to write %s: %s", savePaths[i], err.Error()))
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
