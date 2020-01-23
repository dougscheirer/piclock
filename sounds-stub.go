// +build noaudio

package main

import (
	"log"
)

func init() {
	features = append(features, "noaudio")
}

type noSounds struct {
	playFreqs []string
	playTiming []string
	mp3 string
	loopMp3 bool
}

func (ns noSounds) playIt(sfreqs []string, timing []string, stop chan bool) {
	log.Println("STUB: playIt")
	ns.playFreqs = sfreqs
	ns.playTiming = timing
	// do something about the stop channel, like wait for it
}

func (ns noSounds) playMP3(runtime runtimeConfig, fName string, loop bool, stop chan bool) {
	log.Println("STUB: playMP3 " + fName)
	ns.mp3 = fName
	ns.loopMp3 = loop
	// do something about the stop channel, like wait for it
}
