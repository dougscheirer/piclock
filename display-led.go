package main

import "piclock/sevenseg_backpack"

type ledDisplay struct {
	ssb *sevenseg_backpack.Sevenseg
}

func (ss *ledDisplay) OpenDisplay(settings configSettings) error {
	var err error
	ss.ssb, err = sevenseg_backpack.Open(
		settings.GetByte(sI2CDev),
		settings.GetInt(sI2CBus),
		false)
	return err
}

func (ss *ledDisplay) DebugDump(on bool) {
	ss.ssb.DebugDump(on)
}

func (ss *ledDisplay) SetBrightness(b uint8) error {
	return ss.ssb.SetBrightness(b)
}

func (ss *ledDisplay) DisplayOn(on bool) {
	ss.ssb.DisplayOn(on)
}

func (ss *ledDisplay) Print(e string) error {
	return ss.ssb.Print(e)
}

func (ss *ledDisplay) SetBlinkRate(r uint8) error {
	return ss.ssb.SetBlinkRate(r)
}

func (ss *ledDisplay) RefreshOn(on bool) error {
	return ss.ssb.RefreshOn(on)
}

func (ss *ledDisplay) ClearDisplay() error {
	return ss.ClearDisplay()
}

func (ss *ledDisplay) SegmentOn(pos byte, seg byte, on bool) error {
	return ss.SegmentOn(pos, seg, on)
}
