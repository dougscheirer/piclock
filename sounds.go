// +build !noaudio

package main

import (
	"log"
	"math"
	"os/exec"
	"strconv"
	"time"

	"github.com/bobertlo/go-mpg123/mpg123"
	"github.com/gordonklaus/portaudio"
)

func init() {
	features = append(features, "audio")
}

const sampleRate = 44100

// two functions exist here:
//   PlayPattern(segments []soundSegment, stop chan bool)
//    given a series of frequencies/level/duration, play each in a repeating pattern
//   playMP3(runtime runtimeConfig, file string, stop chan bool)
//    given an MP3 file, play it on repeat

type soundSegment struct {
	frequencies []float64
	duration    time.Duration
	level       float64
	rampDown    time.Duration
}

// this is runtime info for generating the waves
type wave struct {
	step, phase float64
}

// a single segment of sounds, volume, and step information
type playSegment struct {
	steps    int64   // total steps
	level    float64 // volume multiplier
	waves    []wave  // runtime info on the sound
	rampDown int64   // # of steps below which we fade the level
}

type playbackPattern struct {
	*portaudio.Stream
	segments         []playSegment
	curSegment       int
	segmentRemaining int64
}

type realSounds struct {
	dummy int
}

// call this as 'go PlayPattern()'
func playPattern(pattern []soundSegment, stop chan bool) {
	portaudio.Initialize()
	defer portaudio.Terminate()
	s := newPlaySegments(pattern)
	defer s.Close()
	if err := s.Start(); err != nil {
		log.Println(err.Error())
		return
	}

	// block on the stop
	<-stop
	s.Stop()
}

func newPlaySegments(pattern []soundSegment) *playbackPattern {
	// turn pattern into an array of playSegment, stored in a playbackPattern
	var pb playbackPattern
	pb.curSegment = -1

	pb.segments = make([]playSegment, len(pattern))
	for i := range pattern {
		// turn the array of frequencies into a wave array stored in pb.segments[i]
		pb.segments[i].waves = make([]wave, len(pattern[i].frequencies))
		pb.segments[i].level = pattern[i].level
		pb.segments[i].steps = int64(pattern[i].duration * time.Duration(sampleRate) / time.Second)
		pb.segments[i].rampDown = int64(pattern[i].rampDown * time.Duration(sampleRate) / time.Second)
		// calculate the wave steps for each wave
		for w := range pattern[i].frequencies {
			pb.segments[i].waves[w].step = pattern[i].frequencies[w] / sampleRate
			// phase gets reset each time we start the pattern
		}
	}

	var err error
	pb.Stream, err = portaudio.OpenDefaultStream(0, 2, sampleRate, 0, pb.processAudio)
	if err != nil {
		log.Println(err.Error())
		return nil
	}
	return &pb
}

func (g *playbackPattern) segmentInit(seg *playSegment) {
	g.segmentRemaining = seg.steps
	// (?) zero out all the wave phases
	for i := range seg.waves {
		seg.waves[i].phase = 0
	}
}

func (g *playbackPattern) dumpInfo() {
	log.Printf("curSeg: %d, remaining: %d\n", g.curSegment, g.segmentRemaining)
}

func (g *playbackPattern) processAudio(out [][]float32) {
	// g.dumpInfo()
	for i := range out[0] {
		// start the next segment?
		if g.segmentRemaining <= 0 {
			g.curSegment = (g.curSegment + 1) % len(g.segments)
			g.segmentInit(&g.segments[g.curSegment])
		}
		curSeg := &g.segments[g.curSegment]
		g.segmentRemaining--

		// ramp down form normal level to 0 near the end of the segment
		level := curSeg.level
		if g.segmentRemaining < curSeg.rampDown {
			level = level * float64(g.segmentRemaining/curSeg.rampDown)
		}
		// gather the relevant audio level for this segment and time
		var val float32 = 0
		for w := range curSeg.waves {
			val += float32(math.Sin(2*math.Pi*curSeg.waves[w].phase) * level)
			_, curSeg.waves[w].phase = math.Modf(curSeg.waves[w].phase + curSeg.waves[w].step)
		}

		// average out the signal (if any)
		if len(curSeg.waves) > 0 {
			val = val / float32(len(curSeg.waves))
		}

		out[0][i] = val // L
		out[1][i] = val // R
	}
}

func getDecoder(fname string) *mpg123.Decoder {
	decoder, err := mpg123.NewDecoder("")
	if err != nil {
		log.Println(err.Error())
		return nil
	}

	if err = decoder.Open(fname); err != nil {
		log.Println(err.Error())
		return nil
	}

	// get audio format information
	rate, channels, _ := decoder.GetFormat()

	// make sure output format does not change
	decoder.FormatNone()
	decoder.Format(rate, channels, mpg123.ENC_SIGNED_16)

	return decoder
}

func (rs *realSounds) playMP3(runtime runtimeConfig, fName string, loop bool, stop chan bool) {
	// just run mpg123 or the pi fails to play
	cmd := exec.Command("mpg123", fName)
	completed := make(chan error, 1)
	replayMax := 5

	go func() {
		completed <- cmd.Run()
	}()

	stopPlayback := false

	for {
		runtime.rtc.Sleep(100 * time.Millisecond)
		select {
		case <-stop:
			stopPlayback = true
		case done := <-completed:
			log.Printf("%v", done)
			if !loop || replayMax < 0 {
				return
			}
			replayMax--
			log.Println("Replay")
			go func() {
				completed <- cmd.Run()
			}()
		default:
		}
		if stopPlayback {
			log.Println("Stopping playback")
			cmd.Process.Kill()
			return
		}
	}
}

func (rs *realSounds) playIt(sfreqs []string, timing []string, stop chan bool) {
	freqs := make([]float64, 0, len(sfreqs))
	for i := range sfreqs {
		f, e := strconv.ParseFloat(sfreqs[i], 64)
		if e != nil {
			continue
		}
		freqs = append(freqs, f)
	}

	timings := make([]time.Duration, 0, len(timing))
	for i := range timing {
		d, e := time.ParseDuration(timing[i])
		if e != nil {
			continue
		}
		timings = append(timings, d)
	}

	// do 3 runs:
	// 1s on + off for 3s
	// .75 on, .25 off for 3s
	// .95 on, 0.05 off for 3s
	segs := make([]soundSegment, len(timings))

	for i := 0; i < len(segs); i++ {
		segs[i].level = float64((i + 1) % 2)
		segs[i].duration = timings[i]
		segs[i].frequencies = freqs
		segs[i].rampDown = 20 * time.Millisecond
	}

	go playPattern(segs, stop)
}
