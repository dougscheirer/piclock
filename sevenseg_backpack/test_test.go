package sevenseg_backpack

import (
	"fmt"
	"log"
	"runtime"
	"sort"
	"testing"
	"time"

	"gotest.tools/assert"
)

func isSimulated() bool {
	simulated := true
	if runtime.GOARCH == "arm" {
		simulated = false
	}
	return simulated
}

func setup(t *testing.T) *Sevenseg {
	simulated := isSimulated()
	display, err := Open(0x70, 1, simulated) // set to false when on a PI
	display.DebugDump(simulated)

	if err != nil {
		log.Printf("Failed to open: %s\n", err.Error())
		assert.Assert(t, false)
	}

	return display
}

func sleeper(d time.Duration) {
	if isSimulated() {
		return
	}

	time.Sleep(d)
}

func TestBasicDisplay(t *testing.T) {
	// runBasicDisplayImpl(t, true)
	runBasicDisplayImpl(t, false)
}

func runBasicDisplayImpl(t *testing.T, inverted bool) {
	display := setup(t)

	display.ClearDisplay()
	display.SetInverted(inverted)

	// init
	display.Print("8.8.:8.8.")
	sleeper(2 * time.Second)
	// apply brightness levels
	for i := 0; i < 16; i++ {
		display.SetBrightness(uint8(i))
		sleeper(150 * time.Millisecond)
	}
	// try some blink rates
	for i := 1; i < 4; i++ {
		display.SetBlinkRate(uint8(i))
		sleeper(5000 * time.Millisecond)
	}
	// no blink please
	display.SetBlinkRate(0)
	// ramp down brightness
	for i := 15; i >= 0; i-- {
		display.SetBrightness(uint8(i))
		sleeper(150 * time.Millisecond)
	}

	// mid-bright please
	display.SetBrightness(7)

	// turn display on and off
	for i := 0; i < 20; i++ {
		var on = true
		if i%2 == 0 {
			on = false
		}
		display.DisplayOn(on)
		sleeper(250 * time.Millisecond)
	}

	display.DisplayOn(true)
}

func TestCharOutput(t *testing.T) {
	runTestCharOutput(t, false)
	// runTestCharOutput(t, true)
}

func runTestCharOutput(t *testing.T, inverted bool) {
	display := setup(t)

	var knownChars map[byte]byte = digitValues
	if inverted {
		knownChars = inverseDigitValues
	}

	// run the print offset tests
	// get all the keys and sort them
	keys := []int{}
	for k := range knownChars {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)

	i := 0
	for _, v := range keys {
		log.Printf("print '%c'", v)
		display.PrintFromPosition(fmt.Sprintf("%c", v), i%4)
		sleeper(450 * time.Millisecond)
		i++
	}

	// print all the things we know
	for _, v := range keys {
		s := fmt.Sprintf("%c.%c.:%c.%c.", v, v, v, v)
		log.Println(s)
		display.Print(s)
		sleeper(450 * time.Millisecond)
	}

	// test the offset print
	buffer := "test...test...test...done"
	for i:=0;i<len(buffer);i++ {
		display.PrintOffset(buffer, i)
		sleeper(150*time.Millisecond)
	}
}

func TestSegments(t *testing.T) {
	display := setup(t)
	segmentOrder := []byte{
		LED_TOP,
		LED_TOPR,
		LED_BOTR,
		LED_BOT,
		LED_BOTL,
		LED_TOPL,
		LED_MID,
		LED_DECIMAL}

	for j := 0; j < 100; j++ {
		for i := 0; i < len(segmentOrder); i++ {
			display.RefreshOn(false)
			display.ClearDisplay()
			for p := 0; p < 4; p++ {
				display.SegmentOn(byte(p), segmentOrder[i], true)
			}
			display.RefreshOn(true)
			time.Sleep(25 * time.Millisecond)
		}
	}
}
