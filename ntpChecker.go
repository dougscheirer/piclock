package main

import (
	"encoding/json"
	"fmt"
	"time"
)

type ntpChecker struct {
	url string
}

func (ntp *ntpChecker) getIPDateTime(rt runtimeConfig) time.Time {
	// get from url or http://worldtimeapi.org/api/ip
	jsonPath := rt.settings.GetString(sIPTime)
	rt.logger.Printf("Fetching time from " + jsonPath)

	results := make(chan []byte, 20)

	go func() {
		results <- OOBFetch(jsonPath)
	}()

	var f interface{}
	err := json.Unmarshal(<-results, &f)

	if err != nil {
		rt.logger.Printf("Error unmarshalling time: " + err.Error())
		return time.Time{}
	}

	itemsMap := f.(map[string]interface{})
	layout := "2006-01-02T15:04:05.999999-07:00"
	t, e := time.Parse(layout, fmt.Sprintf("%v", itemsMap["datetime"]))

	if e != nil {
		rt.logger.Printf("Error parsing time: " + e.Error())
		return time.Time{}
	}

	return t
}
