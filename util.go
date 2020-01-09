// utility functions
package main

import "time"

type commChannels struct {
	quit    chan struct{}
	alarms  chan checkMsg
	effects chan effect
	loader  chan loaderMsg
}

type clock interface {
	now() time.Time
}

type rtc struct {
}

type wallClock struct {
	curTime time.Time
}

func (r rtc) now() time.Time {
	return time.Now()
}

func (w wallClock) now() time.Time {
	return w.curTime
}

type runtimeConfig struct {
	comms     commChannels
	rtc       clock
	wallClock clock
}

func initCommChannels() commChannels {
	quit := make(chan struct{}, 1)
	alarmChannel := make(chan checkMsg, 1)
	effectChannel := make(chan effect, 1)
	loaderChannel := make(chan loaderMsg, 1)

	// wait on our workers:
	// alarm fetcher
	// clock runner (effects)
	// alarm checker
	// button checker
	wg.Add(4)

	return commChannels{quit: quit, alarms: alarmChannel, effects: effectChannel, loader: loaderChannel}
}

func initRuntime(rtc clock, wallClock clock) runtimeConfig {
	return runtimeConfig{
		rtc:       rtc,
		wallClock: wallClock,
		comms:     initCommChannels()}
}
