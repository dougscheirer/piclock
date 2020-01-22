// +build nolcd

package main

import "log"

type sevensegShim struct {
	i2cBus     int
	curDisplay string
	debugDump  bool
	brightness uint8
	displayOn  bool
	blinkRate  uint8
	refreshOn  bool
	segments   [8][8]bool
}

func openDisplay(settings *configSettings) (*sevensegShim, error) {
	this := &sevensegShim{
		i2cBus:     settings.GetInt("i2cBus"),
		curDisplay: "",
		debugDump:  settings.GetBool("debugDump"),
		brightness: 0,
		displayOn:  false,
		blinkRate:  0,
		refreshOn:  false,
	}
	return this, nil
}

func (this *sevensegShim) DebugDump(on bool) error {
	this.debugDump = on
	return nil
}

func (this *sevensegShim) SetBrightness(b uint8) error {
	this.brightness = b
	return nil
}

func (this *sevensegShim) DisplayOn(on bool) error {
	this.displayOn = on
	return nil
}

func (this *sevensegShim) Print(e string) error {
	if e != this.curDisplay {
		log.Print(e)
	}
	this.curDisplay = e
	return nil
}

func (this *sevensegShim) SetBlinkRate(r uint8) error {
	this.blinkRate = r
	return nil
}

func (this *sevensegShim) RefreshOn(on bool) error {
	this.refreshOn = on
	return nil
}

func (this *sevensegShim) ClearDisplay() error {
	this.curDisplay = ""
	return nil
}

func (this *sevensegShim) SegmentOn(pos byte, seg byte, on bool) error {
	this.curDisplay = ""
	this.segments[pos][seg] = on
	// debug output?
	log.Printf("%d/%d set to %v", pos, seg, on)
	return nil
}
