package main

import (
	"fmt"
	"log"
	"piclock/sevenseg_backpack"
)

type logDisplay struct {
	i2cBus      int
	curDisplay  string
	debugDump   bool
	brightness  uint8
	displayOn   bool
	blinkRate   uint8
	refreshOn   bool
	segments    [8][8]bool
	audit       []string
	auditErrors []error
	ssb         *sevenseg_backpack.Sevenseg
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
	ld.auditErrors = []error{}
	// open the display in simulated mode
	ld.ssb, _ = sevenseg_backpack.Open(0, 0, true)

	return nil
}

func (ld *logDisplay) DebugDump(on bool) {
	ld.debugDump = on
	ld.ssb.DebugDump(on)
}

func (ld *logDisplay) SetBrightness(b uint8) error {
	ld.brightness = b
	err := ld.ssb.SetBrightness(b)
	if err != nil {
		ld.auditErrors = append(ld.auditErrors, err)
	}
	return err
}

func (ld *logDisplay) DisplayOn(on bool) {
	ld.displayOn = on
	ld.ssb.DisplayOn(on)
}

func (ld *logDisplay) Print(e string) error {
	if e != ld.curDisplay {
		log.Println(e)
		ld.audit = append(ld.audit, e)
	}
	ld.curDisplay = e
	err := ld.ssb.Print(e)
	if err != nil {
		ld.auditErrors = append(ld.auditErrors, err)
	}
	return err
}

func (ld *logDisplay) PrintOffset(e string, offset int) (string, error) {
	if e != ld.curDisplay {
		log.Printf("%s / %d", e, offset)
	}
	cur, err := ld.ssb.PrintOffset(e, offset)
	ld.curDisplay = cur
	ld.audit = append(ld.audit, cur)
	if err != nil {
		ld.auditErrors = append(ld.auditErrors, err)
	}
	return cur, err
}

func (ld *logDisplay) SetBlinkRate(r uint8) error {
	ld.blinkRate = r
	err := ld.ssb.SetBlinkRate(r)
	if err != nil {
		ld.auditErrors = append(ld.auditErrors, err)
	}
	return err
}

func (ld *logDisplay) RefreshOn(on bool) error {
	ld.refreshOn = on
	err := ld.ssb.RefreshOn(on)
	if err != nil {
		ld.auditErrors = append(ld.auditErrors, err)
	}
	return err
}

func (ld *logDisplay) ClearDisplay() {
	ld.curDisplay = ""
	ld.ssb.ClearDisplay()
}

func (ld *logDisplay) SegmentOn(pos byte, seg byte, on bool) error {
	ld.curDisplay = ""
	ld.segments[pos][seg] = on
	// debug output?
	log.Printf("%d/%d set to %v\n", pos, seg, on)
	ld.audit = append(ld.audit, fmt.Sprintf("seg %d/%d %v", pos, seg, on))
	err := ld.ssb.SegmentOn(pos, seg, on)
	if err != nil {
		ld.auditErrors = append(ld.auditErrors, err)
	}
	return err
}
