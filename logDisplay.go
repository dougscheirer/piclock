package main

import (
	"fmt"
	"log"
)

type logDisplay struct {
	i2cBus     int
	curDisplay string
	debugDump  bool
	brightness uint8
	displayOn  bool
	blinkRate  uint8
	refreshOn  bool
	segments   [8][8]bool
	audit      []string
}

func (ld *logDisplay) OpenDisplay(settings configSettings) error {
	ld.i2cBus = settings.GetInt(sI2CBus)
	ld.curDisplay = ""
	ld.debugDump = settings.GetBool(sDebug)
	ld.brightness = 0
	ld.displayOn = false
	ld.blinkRate = 0
	ld.refreshOn = false
	ld.audit = []string{}
	return nil
}

func (ld *logDisplay) DebugDump(on bool) {
	ld.debugDump = on
}

func (ld *logDisplay) SetBrightness(b uint8) error {
	ld.brightness = b
	return nil
}

func (ld *logDisplay) DisplayOn(on bool) {
	ld.displayOn = on
}

func (ld *logDisplay) Print(e string) error {
	if e != ld.curDisplay {
		log.Println(e)
		ld.audit = append(ld.audit, e)
	}
	ld.curDisplay = e
	return nil
}

func (ld *logDisplay) SetBlinkRate(r uint8) error {
	ld.blinkRate = r
	return nil
}

func (ld *logDisplay) RefreshOn(on bool) error {
	ld.refreshOn = on
	return nil
}

func (ld *logDisplay) ClearDisplay() error {
	ld.curDisplay = ""
	return nil
}

func (ld *logDisplay) SegmentOn(pos byte, seg byte, on bool) error {
	ld.curDisplay = ""
	ld.segments[pos][seg] = on
	// debug output?
	log.Printf("%d/%d set to %v\n", pos, seg, on)
	ld.audit = append(ld.audit, fmt.Sprintf("seg %d/%d %v", pos, seg, on))
	return nil
}
