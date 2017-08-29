package main

import (
	"fmt"
)

var displayBuf [5]uint8;

// open an i2c connection
func NewI2C(addr uint8, bus int) (*I2C, error) {
	f, err := os.OpenFile(fmt.Sprintf("/dev/i2c-%d", bus), os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err := ioctl(f.Fd(), I2C_SLAVE, uintptr(addr)); err != nil {
		return nil, err
	}
	this := &I2C{rc: f}
	return this, nil
}

func main() {
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