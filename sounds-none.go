// +build noaudio

package main

func init() {
	features = append(features, "noaudio")
}

// just make realSounds the same as nothing
type realSounds struct {
	noSounds
}
