// +build noleds

package main

import "log"

func errorLED(on bool) {
	log.Printf("Set LED 16 to %v", on)
}
