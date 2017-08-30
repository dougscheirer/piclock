package main

import (
	"fmt"
	"piclock/i2c"
	"time"
)

// set a segment on a digit
func setSegment(pos byte, segNum byte, display [10]uint8) [10]uint8 {
	if pos > 2 {
		pos++
	}
	// +1 to skip the address byte, & 0x80 to keep the dot
	display[1+pos*2] = (display[1+pos*2] & 0x80) | (1 << segNum)
	return display
}

// turn a dot on or off
func setDot(pos byte, dotOn bool, display [10]uint8) [10]uint8 {
	if pos > 2 {
		pos++
	}
	val := 0x80
	if !dotOn {
		val = 0x00
	}
	display[1+pos*2] = (display[1+pos*2] & 0x7F) | uint8(val)
	return display
}

func setColon(colonOn bool, display [10]uint8) [10]uint8 {
	val := 1
	if !colonOn {
		val = 0
	}
	display[5] = uint8(val)
	return display
}

func main() {

	dev, err := i2c.Open(0x70, 0)
	if err != nil {
		fmt.Printf("Failed to open: %s", err.Error())
		return
	}

	// first some commands
  dev.WriteByte(0x21)  // turn on oscillator
  dev.WriteByte(0x81)	// turn on display, no blinking
  dev.WriteByte(0xEF)	// max brightness

	// write to display:
	// AA D0 xx D1 xx CL xx D2 xx D3 xx xx xx xx xx xx
	// 0  1  2  3  4  5  6  7  8  9  A  B  C  D  E  F
	// digit bit order:
	//  -    0
	// | |  6  1
	//  -			5
	// | |  4  2
	//  - .  3    7
	//
	//  :  0
	var displayBuf [10]uint8; // every other byte is unused
	for i:=0;i<len(displayBuf);i++ {
		displayBuf[i]=0
	}

	// roll through each digit's 7 bits, flash the dots and flash the colon
	for true {
		for i:=0;i<7;i++ {
			itsOn := false
			if i % 2 == 0 {
				itsOn = true
			}
			for j:=0;j<4;j++ {
				displayBuf = setSegment(byte(j), byte(i), displayBuf)
				displayBuf = setDot(byte(j), itsOn, displayBuf)
			}
			displayBuf = setColon(itsOn, displayBuf)
			dev.Write(displayBuf[:])
			time.Sleep(250 * time.Millisecond)
		}
	}
}
