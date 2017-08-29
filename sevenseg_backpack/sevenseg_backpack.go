package sevenseg_backpack

import "i2c"

digitValues := map[byte]byte {
    ' ': 0x00,
    '-': 0x40,
    '_': 0x08,
    '0': 0x3F,
    '1': 0x06,
    '2': 0x5B,
    '3': 0x4F,
    '4': 0x66,
    '5': 0x6D,
    '6': 0x7D,
    '7': 0x07,
    '8': 0x7F,
    '9': 0x6F,
    'A': 0x77,
    'B': 0x7C,
    'C': 0x39,
    'D': 0x5E,
    'E': 0x79,
    'F': 0x71
}

inverseDigitValues := map[byte]byte {
    ' ': 0x00,
    '-': 0x40,
    '_': 0x08, // TODO: TBD
    '0': 0x3F,
    '1': 0x30,
    '2': 0x5B,
    '3': 0x79,
    '4': 0x74,
    '5': 0x6D,
    '6': 0x6F,
    '7': 0x38,
    '8': 0x7F,
    '9': 0x7D,
    'A': 0x7E,
    'B': 0x67,
    'C': 0x0F,
    'D': 0x73,
    'E': 0x4F,
    'F': 0x4E
}

type Sevenseg struct {
	display [5]byte
	i2c_dev I2C
}

func NewSevenseg(address uint8, bus int) (*Sevenseg, error) {
	i2c, err = i2c.Open(address, bus)
	if err != nil {
		return nil, err
	}
	this := &Sevenseg{i2c_dev: i2c}
	// clear the display
	for i:=0;i<len(display);i++ {
		display[i]=0
	}
	return this, nil
}

func (this *sevenseg) refresh_display()
{
	i2c_dev.Write_byte()
}

func (this *sevenseg) Write_colon(on bool) error {
	// TODO: figure out colon position
	return nil
}