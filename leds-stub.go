// +build noleds

package main

import "log"

func init() {
	features = append(features, "noleds")
}

func setLED(pinNum int, on bool) {
	log.Printf("Set LED %v to %v", pinNum, on)
}
