package main

import (
	"github.com/stianeikeland/go-rpio"
	"google.golang.org/api/calendar/v3"
)

type sounds interface {
	playIt(sfreqs []string, timing []string, stop chan bool)
	playMP3(runtime runtimeConfig, fName string, loop bool, stop chan bool)
}

type buttons interface {
	readButtons(runtime runtimeConfig) (map[string]rpio.State, error)
	setupButtons(pins map[string]buttonMap, runtime runtimeConfig) error
	initButtons(settings configSettings) error
	closeButtons()
	getButtons() *map[string]button
}

type display interface {
	OpenDisplay(settings configSettings) error
	DebugDump(on bool)
	SetBrightness(b uint8) error
	DisplayOn(on bool)
	Print(e string) error
	SetBlinkRate(r uint8) error
	RefreshOn(on bool) error
	ClearDisplay() error
	SegmentOn(pos byte, seg byte, on bool) error
}

type led interface {
	init()
	set(pin int, on bool)
	on(pin int)
	off(pin int)
}

type events interface {
	fetch(runtime runtimeConfig) (*calendar.Events, error)
	getCalendarService(settings configSettings, prompt bool) (*calendar.Service, error)
}
