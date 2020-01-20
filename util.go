// utility functions
package main

import "time"

type commChannels struct {
	quit    chan struct{}
	alarms  chan checkMsg
	effects chan displayEffect
	loader  chan loaderMsg
	leds    chan ledEffect
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
	effectChannel := make(chan displayEffect, 1)
	loaderChannel := make(chan loaderMsg, 1)
	leds := make(chan ledEffect, 1)

	return commChannels{
		quit:    quit,
		alarms:  alarmChannel,
		effects: effectChannel,
		loader:  loaderChannel,
		leds:    leds}
}

func initRuntime(rtc clock, wallClock clock) runtimeConfig {
	return runtimeConfig{
		rtc:       rtc,
		wallClock: wallClock,
		comms:     initCommChannels()}
}
