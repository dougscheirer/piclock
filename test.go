package main

import (
	"fmt"
	"piclock/sevenseg_backpack"
	"time"
)

func main() {

	display, err := sevenseg_backpack.Open(0x70, 0)
	if err != nil {
		fmt.Printf("Failed to open: %s\n", err.Error())
		return
	}

	display.ClearDisplay()

	segmentOrder := []byte{
		sevenseg_backpack.LED_TOP,
		sevenseg_backpack.LED_TOPR,
		sevenseg_backpack.LED_BOTR,
		sevenseg_backpack.LED_BOT,
		sevenseg_backpack.LED_BOTL,
		sevenseg_backpack.LED_TOPL }

	for true {
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