package sevenseg_backpack

import "i2c"

// commands we support

// OSC on/off 0/1
i2c_OSC_CMD   := 0x20
i2c_OSC_ON    := 0x21
i2c_OSC_OFF   := 0x20

// display on/off and 2 "blink" bits in position 2+1
i2c_DISPLAY_CMD := 0x80
i2c_DISPLAY_ON  := 0x81
i2c_DISPLAY_OFF := 0x80

i2c_BLINK_OFF := 0
i2c_BLINK_2HZ := 1
i2c_BLINK_1HZ := 2
i2c_BLINK_HALFHZ := 3

// 0x0 -> 0xF brightness levels
i2c_BRIGHTNESS_CMD  := 0xE0
i2c_BRIGHTNESS_MAX  := 0xEF
i2c_BRIGHTNESS_MIN  := 0xE0
i2c_BRIGHTNESS_HALF := 0xE7

// translate characters to bitmasks
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
	display [5*2]byte  // 7-seg skips bytes for each display element
	i2c_dev I2C
}

func NewSevenseg(address uint8, bus int) (*Sevenseg, error) {
	i2c_dev, err = i2c.Open(address, bus)
	if err != nil {
		return nil, err
	}
	this := &Sevenseg{i2c_dev: i2c_dev}
	// clear the display
	for i:=0;i<len(this.display);i++ {
		this.display[i]=0
	}
    // turn on the oscillator, set default brightness
    i2c_dev.WriteByte(i2c_OSC_ON)
    i2c_dev.WriteByte(i2c_BRIGHTNESS_MAX)

    // you still need to call DisplayOn(true) to see stuff
	return this, nil
}

func (this *sevenseg) ClearDisplay() {
    for i:=0;i<len(display);i++ {
        display[i]=0
    }
    
}

func (this *sevenseg) refresh_display()
{
    // start with the address (0)
    buf := [1+len(this.display)]byte
    buf[0] = 0
    for i:=0;i<len(this.display);i++ {
        buf[1+i]=display[i]
    }
	this.i2c_dev.Write(buf)
}

func (this *sevenseg) Write_colon(on bool) error {
	// TODO: figure out colon position
	return nil
}