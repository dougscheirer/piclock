package main

import (
	"log"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/stianeikeland/go-rpio"
)

func main() {
	// BUTTON is the pin number
	// RUNPROG is the thing to run
	pinS, pinSE := os.LookupEnv("BUTTON")
	progS, progSE := os.LookupEnv("RUNPROG")
	_, dir := os.LookupEnv("PULLUP")

	if !pinSE || !progSE {
		log.Fatalf("Must provide a BUTTON and RUNPROG in the environment: %s : %s\n", pinS, progS)
		return
	}
	pin, pinE := strconv.ParseInt(pinS, 0, 64)
	if pinE != nil {
		log.Fatalf("%s is not a number", pinS)
	}
	// open the button for read
	err := rpio.Open()
	if err != nil {
		log.Fatal(err.Error())
	}
	rpioPin := rpio.Pin(pin)
	// for now we only care about the "low" state
	rpioPin.Input() // Input mode
	var pressState rpio.State
	if dir {
		rpioPin.PullUp() // GND => button press
		pressState = rpio.Low
	} else {
		rpioPin.PullDown() // +V -> button press
		pressState = rpio.High
	}

	log.Printf("Watching %v for %v", pin, dir)
	for true {
		s := rpioPin.Read()
		if s == pressState {
			// run the program
			log.Printf("Running %s\n", progS)
			cmd := exec.Command(progS)
			// wait for it to exit
			out, err := cmd.Output()
			if err != nil {
				log.Println(err.Error())
			}
			log.Printf("%s",out)
			// take a nap after running the command
			log.Printf("Sleeping...")
			time.Sleep(5 * time.Second)
		}
		time.Sleep(30 * time.Millisecond)
	}
}

