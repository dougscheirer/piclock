package main

import (
	"runtime"
	"time"
	"fmt"
	"log"
	"piclock/sevenseg_backpack"
)

func main() {
	simulated := true
	if runtime.GOARCH == "arm" {
		simulated = false
	}

	display, err := sevenseg_backpack.Open(0x70, 0, simulated)	// set to false when on a PI
	display.DebugDump(simulated)

	if err != nil {
		log.Printf("Failed to open: %s\n", err.Error())
		return
	}

	runTests(display, false)
	runTests(display, true)
}

func runTests(display *sevenseg_backpack.Sevenseg, inverted bool) {
	display.ClearDisplay()
	display.SetInverted(inverted)

	knownChars := []byte{ ' ', '-', '_', '0','1','2','3','4','5','6','7','8','9',
												'A','B','C','c','D','d','E','F','R','r','H','h','L','l','X'}

	// init
	display.Print("8.8.:8.8.")
	time.Sleep(2 * time.Second)
	// apply brightness levels
	for i:=0;i<16;i++ {
		display.SetBrightness(uint8(i))
		time.Sleep(450*time.Millisecond)
	}
	// try some blink rates
	for i:=1;i<4;i++ {
		display.SetBlinkRate(uint8(i))
		time.Sleep(5000*time.Millisecond)
	}
	// no blink please
	display.SetBlinkRate(0)
	// ramp down brightness
	for i:=15;i>=0;i-- {
		display.SetBrightness(uint8(i))
		time.Sleep(450*time.Millisecond)
	}

	// mid-bright please
	display.SetBrightness(7)

	// turn display on and off
	for i:=0;i<20;i++ {
		var on = true
		if i % 2 == 0 {
			on = false
		}
		display.DisplayOn(on)
		time.Sleep(250 * time.Millisecond)
	}

	display.DisplayOn(true)

	// run the print offset tests
	for i:=0;i<len(knownChars);i++ {
		log.Printf("print '%c'", knownChars[i])
		display.PrintOffset(fmt.Sprintf("%c", knownChars[i]), i % 4)
		time.Sleep(450*time.Millisecond)
	}
	// now in reverse
	for i:=len(knownChars)-1;i>=0;i-- {
		log.Printf("print '%c'", knownChars[i])
		display.PrintOffset(fmt.Sprintf("%c", knownChars[i]), i % 4)
		time.Sleep(450*time.Millisecond)
	}

	// print all the things we know
	for i:=0;i<len(knownChars);i++ {
		c := knownChars[i];
		s := fmt.Sprintf("%c.%c.:%c.%c.", c,c,c,c)
		log.Println(s)
		display.Print(s)
		time.Sleep(450 * time.Millisecond)
	}
	for i:=999;i>=-999;i-- {
		display.Print(fmt.Sprintf("%d",i))
		time.Sleep(25*time.Millisecond)
	}

	segmentOrder := []byte{
		sevenseg_backpack.LED_TOP,
		sevenseg_backpack.LED_TOPR,
		sevenseg_backpack.LED_BOTR,
		sevenseg_backpack.LED_BOT,
		sevenseg_backpack.LED_BOTL,
		sevenseg_backpack.LED_TOPL,
		sevenseg_backpack.LED_MID,
		sevenseg_backpack.LED_DECIMAL }

	for j:=0;j<100;j++ {
		for i:=0;i<len(segmentOrder);i++ {
			display.RefreshOn(false)
			display.ClearDisplay()
			for p:=0;p<4;p++ {
				display.SegmentOn(byte(p), segmentOrder[i], true)
			}
			display.RefreshOn(true)
			time.Sleep(25 * time.Millisecond)
		}
	}
}
