// +build !noaudio

package main

import (
	"os/exec"
)

func init() {
	features = append(features, "audio")
}

const sampleRate = 44100

type realSounds struct {
}

func (rs *realSounds) playIt(rt runtimeConfig, sfreqs []string, timing []string, stop chan bool, done chan bool) {
	// TODO: make this work without pulseaudio
	rt.logger.Printf("playIt is not really implemented")
}

func (rs *realSounds) playMP3(rt runtimeConfig, fName string, loop bool, stop chan bool, done chan bool) {
	go rs.playMP3Later(rt, fName, loop, stop, done)
}

func (rs *realSounds) playMP3Later(rt runtimeConfig, fName string, loop bool, stop chan bool, done chan bool) {
	// when we exit the function, tell someone that we're done
	defer func() {
		done <- true
	}()

	// TODO: make just the launch part of the interface impl, move the loop
	//   logic somewhere else so we can test it

	// just run mpg123 or the pi fails to play
	cmd := exec.Command("mpg123", fName)
	completed := make(chan error, 1)
	// TODO: make configurable?
	replayMax := 5

	go func() {
		completed <- cmd.Run()
	}()

	stopPlayback := false

	for {
		rt.clock.Sleep(dAlarmSleep)
		select {
		case <-stop:
			stopPlayback = true
		case done := <-completed:
			rt.logger.Printf("%v", done)
			if !loop || replayMax < 0 {
				return
			}
			replayMax--
			rt.logger.Println("Replay")
			cmd = exec.Command("mpg123", fName)
			go func() {
				completed <- cmd.Run()
			}()
		default:
		}
		if stopPlayback {
			rt.logger.Println("Stopping playback")
			cmd.Process.Kill()
			return
		}
	}
}
