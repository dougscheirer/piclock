// +build nolcd

package main

type sevensegShim struct {
	i2cBus int
}

func openDisplay(settings *configSettings) (*sevensegShim, error) {
	this := &sevensegShim{
		i2cBus: settings.GetInt("i2cBus")}
	return this, nil
}

func (this *sevensegShim) DebugDump(on bool) error {
	return nil
}

func (this *sevensegShim) SetBrightness(b uint8) error {
	return nil
}

func (this *sevensegShim) DisplayOn(on bool) error {
	return nil
}

func (this *sevensegShim) Print(e string) error {
	return nil
}

func (this *sevensegShim) SetBlinkRate(r uint8) error {
	return nil
}

func (this *sevensegShim) RefreshOn(on bool) error {
	return nil
}

func (this *sevensegShim) ClearDisplay() error {
	return nil
}

func (this *sevensegShim) SegmentOn(pos byte, seg byte, on bool) error {
	return nil
}
