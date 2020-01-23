package main

type sounds interface {
	playIt(sfreqs []string, timing []string, stop chan bool)
	playMP3(runtime runtimeConfig, fName string, loop bool, stop chan bool)
}
