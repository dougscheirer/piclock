// +build noleds

package main

import "log"

func init() {
	features = append(features, "noleds")
}

func errorLED(on bool) {
	log.Printf("Set LED 16 to %v", on)
}
