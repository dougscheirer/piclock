package main

import (
	"fmt"
	"log"
	"time"

	"google.golang.org/api/calendar/v3"
)

func (ge *gcalEvents) fetch(runtime runtimeConfig) (*calendar.Events, error) {
	settings := runtime.settings
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
	t := runtime.rtc.Now().Format(time.RFC3339)
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
