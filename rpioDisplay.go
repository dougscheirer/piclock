package main

import "dscheirer.com/piclock/sevenseg_backpack"

type rpioDisplay struct {
	ssb *sevenseg_backpack.Sevenseg
}

func (ss *rpioDisplay) OpenDisplay(settings configSettings) error {
	var err error
	ss.ssb, err = sevenseg_backpack.Open(
		settings.GetByte(sI2CDev),
		settings.GetInt(sI2CBus),
		false)
	// by default, turn it on?
	ss.DisplayOn(true)
	return err
}

func (ss *rpioDisplay) DebugDump(on bool) {
	ss.ssb.DebugDump(on)
}

func (ss *rpioDisplay) SetBrightness(b uint8) error {
	return ss.ssb.SetBrightness(b)
}

func (ss *rpioDisplay) DisplayOn(on bool) {
	ss.ssb.DisplayOn(on)
}

func (ss *rpioDisplay) Print(e string) error {
	return ss.ssb.Print(e)
}

func (ss *rpioDisplay) PrintOffset(e string, offset int) (string, error) {
	return ss.ssb.PrintOffset(e, offset)
}

func (ss *rpioDisplay) SetBlinkRate(r uint8) error {
	return ss.ssb.SetBlinkRate(r)
}

func (ss *rpioDisplay) RefreshOn(on bool) error {
	return ss.ssb.RefreshOn(on)
}

func (ss *rpioDisplay) ClearDisplay() {
	ss.ClearDisplay()
}

func (ss *rpioDisplay) SegmentOn(pos byte, seg byte, on bool) error {
	return ss.SegmentOn(pos, seg, on)
}
