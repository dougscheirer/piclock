// utility functions
package main

import "time"

type commChannels struct {
	quit     chan struct{}
	alarms   chan almStateMsg
	effects  chan displayEffect
	almState chan almStateMsg
	leds     chan ledEffect
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
	alarmChannel := make(chan almStateMsg, 10)
	effectChannel := make(chan displayEffect, 10)
	loaderChannel := make(chan almStateMsg, 10)
	leds := make(chan ledEffect, 10)

	return commChannels{
		quit:     quit,
		alarms:   alarmChannel,
		effects:  effectChannel,
		almState: loaderChannel,
		leds:     leds}
}

func initRuntime(rtc clock, wallClock clock) runtimeConfig {
	return runtimeConfig{
		rtc:       rtc,
		wallClock: wallClock,
		comms:     initCommChannels()}
}
