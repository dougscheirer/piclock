// +build noaudio

package main

import (
	"log"
)

func init() {
	features = append(features, "noaudio")
}

func playIt(sfreqs []string, timing []string, stop chan bool) {
	log.Println("STUB: playIt")
}

func PlayMP3(fName string, loop bool, stop chan bool) {
	log.Println("STUB: PlayMP3 " + fName)
}
