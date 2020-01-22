// +build test

package main

import "google.golang.org/api/calendar/v3"

func fetchEventsFromCalendar(settings *configSettings, runtime runtimeConfig) (*calendar.Events, error) {
	// TODO: use a faked list of events
	return nil, nil
}

// stubbed out so that main.go will build
func getCalendarService(settings *configSettings, prompt bool) (*calendar.Service, error) {
	return nil, nil
}
