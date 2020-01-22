// +build !nolcd

package main

import "piclock/sevenseg_backpack"

type sevensegShim struct {
	ssb *sevenseg_backpack.Sevenseg
}

func openDisplay(settings *configSettings) (*sevensegShim, error) {
	ssb, err := sevenseg_backpack.Open(
		settings.GetByte("i2cDevice"),
		settings.GetInt("i2cBus"),
		settings.GetBool("i2cSimulated"))
	return &sevensegShim{ssb: ssb}, err
}

func (this *sevensegShim) DebugDump(on bool) {
	this.ssb.DebugDump(on)
}

func (this *sevensegShim) SetBrightness(b uint8) error {
	return this.ssb.SetBrightness(b)
}

func (this *sevensegShim) DisplayOn(on bool) {
	this.ssb.DisplayOn(on)
}

func (this *sevensegShim) Print(e string) error {
	return this.ssb.Print(e)
}

func (this *sevensegShim) SetBlinkRate(r uint8) error {
	return this.ssb.SetBlinkRate(r)
}

func (this *sevensegShim) RefreshOn(on bool) error {
	return this.ssb.RefreshOn(on)
}

func (this *sevensegShim) ClearDisplay() error {
	return this.ClearDisplay()
}

func (this *sevensegShim) SegmentOn(pos byte, seg byte, on bool) error {
	return this.SegmentOn(pos, seg, on)
}
