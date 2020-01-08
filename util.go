// utility functions
package main

import "time"

type CommChannels struct {
	quit    chan struct{}
	alarms  chan CheckMsg
	effects chan Effect
	loader  chan LoaderMsg
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

type RuntimeConfig struct {
	comms     CommChannels
	rtc       clock
	wallClock clock
}

func initCommChannels() CommChannels {
	quit := make(chan struct{}, 1)
	alarmChannel := make(chan CheckMsg, 1)
	effectChannel := make(chan Effect, 1)
	loaderChannel := make(chan LoaderMsg, 1)

	// wait on our workers:
	// alarm fetcher
	// clock runner (effects)
	// alarm checker
	// button checker
	wg.Add(4)

	return CommChannels{quit: quit, alarms: alarmChannel, effects: effectChannel, loader: loaderChannel}
}

func initRuntime(rtc clock, wallClock clock) RuntimeConfig {
	return RuntimeConfig{
		rtc:       rtc,
		wallClock: wallClock,
		comms:     initCommChannels()}
}
