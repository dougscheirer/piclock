package main

import (
	"fmt"
	"piclock/i2c"
	"time"
)

func main() {

	dev, err := i2c.Open(0x70, 0)
	if err != nil {
		fmt.Printf("Failed to open: %s", err.Error())
	} else {
		var buf [16]uint8;
		for j:=1;j<11;j+=2{
			fmt.Printf("j is %d\n",j)
			for b:=0;b<len(buf);b++ { buf[b]=0 }
			for i:=0;i<256;i++ {
				buf[j] = uint8(i)
				dev.Write(buf[:])
				time.Sleep(25*time.Millisecond)
			}
		}
	}
	var displayBuf [5]byte
	for i := 0; i < 5; i++ {
		displayBuf[i] = 0
	}
	for i := 0; i < 34; i++ {
		// find the bit to shift (i/8)
		var pos int = i/8
		var bit uint = 7-uint(i%8)
		if bit == 7 && pos > 0 {
			// clear previous byte
			displayBuf[pos-1] = 0
		}
		// set bit in displayBuf
		displayBuf[pos] = 1 << bit
		fmt.Printf("%d : %d : ", pos, bit)
		for d := 0; d < 5; d++ {
			for b := 7; b >= 0; b-- {
				val := (displayBuf[d] & (1 << uint(b)))
				if val != 0 {
					val = 1
				}
				fmt.Printf("%d", val)
			}
		}
		fmt.Printf("\n")
	}
}
