package main

import (
	"github.com/stianeikeland/go-rpio"
	"google.golang.org/api/calendar/v3"
)

type sounds interface {
	playIt(rt runtimeConfig, sfreqs []string, timing []string, stop chan bool, done chan bool)
	playMP3(rt runtimeConfig, fName string, loop bool, stop chan bool, done chan bool)
}

type buttons interface {
	readButtons(rt runtimeConfig) (map[string]rpio.State, error)
	setupButtons(pins map[string]buttonMap, rt runtimeConfig) error
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
	fetch(rt runtimeConfig) (*calendar.Events, error)
	getCalendarService(settings configSettings, prompt bool) (*calendar.Service, error)
	loadAlarms(rt runtimeConfig, loadID int, report bool)
	downloadMusicFiles(settings configSettings, cE chan displayEffect)
	generateSecret(rt runtimeConfig) string
}

type configService interface {
	launch(handler *apiHandler, addr string)
	stop()
}
