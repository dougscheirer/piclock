package main

import (
	"errors"
	"fmt"
	"log"

	"google.golang.org/api/calendar/v3"
)

type testEvents struct {
	events      calendar.Events
	errorResult bool
	errorCount  int
	fetches     int
}

const infinite int = -2

func (te *testEvents) setFails(cnt int) {
	if cnt <= 0 {
		cnt = infinite
	}
	te.errorCount = cnt
	te.errorResult = true
}

func (te *testEvents) fetch(rt runtimeConfig) (*calendar.Events, error) {
	te.fetches++
	log.Printf("Fetch: %d", te.fetches)
	if te.errorResult {
		err := errors.New("Bad fetch error")
		if te.errorCount != infinite {
			if te.errorCount > 0 {
				te.errorCount--
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	// use a faked list of events
	var events calendar.Events
	events.Items = make([]*calendar.Event, 5)
	for k := range events.Items {
		events.Items[k] = &calendar.Event{}
		events.Items[k].Start = &calendar.EventDateTime{DateTime: fmt.Sprintf("2020-01-01T0%d:00:00.00Z", k)}
		events.Items[k].Id = fmt.Sprintf("%d", k)
		switch k % 3 {
		case 0:
			events.Items[k].Summary = "music"
		case 1:
			events.Items[k].Summary = "tone"
		case 2:
			events.Items[k].Summary = "music dance"
		default:
			events.Items[k].Summary = "n/a"
		}
	}
	return &events, nil
}

func (te *testEvents) getCalendarService(settings configSettings, prompt bool) (*calendar.Service, error) {
	return nil, nil
}

func (te *testEvents) downloadMusicFiles(settings configSettings, display chan displayEffect) {
	// note that we got a call to do this?
}

func (te *testEvents) loadAlarms(rt runtimeConfig, loadID int, report bool) {
	// do the thing in realtime for testing
	loadAlarmsImpl(rt, loadID, report)
}
