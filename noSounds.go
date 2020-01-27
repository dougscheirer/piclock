package main

import (
	"log"
)

type noSounds struct {
	playFreqs  []string
	playTiming []string
	mp3        string
	loopMp3    bool
	playItCnt  int
	playMP3Cnt int
	done       chan bool
}

func (ns *noSounds) playIt(rt runtimeConfig, sfreqs []string, timing []string, stop chan bool, done chan bool) {
	log.Println("STUB: playIt")
	ns.playFreqs = sfreqs
	ns.playTiming = timing
	ns.done = done
	// pretend we did this
	ns.playItCnt++
}

func (ns *noSounds) playMP3(rt runtimeConfig, fName string, loop bool, stop chan bool, done chan bool) {
	log.Println("STUB: playMP3 " + fName)
	ns.mp3 = fName
	ns.loopMp3 = loop
	ns.done = done
	// pretend we did this
	ns.playMP3Cnt++
}
