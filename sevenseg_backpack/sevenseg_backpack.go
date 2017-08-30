package sevenseg_backpack

import (
    "piclock/i2c"
    "strings"
    "errors"
    "fmt"
    )

// commands we support
// OSC on/off 0/1
const i2c_OSC_CMD   = 0x20
const i2c_OSC_ON    = 0x21
const i2c_OSC_OFF   = 0x20

// display on/off and 2 "blink" bits in position 2+1
const i2c_DISPLAY_CMD = 0x80
const i2c_DISPLAY_ON  = 0x81
const i2c_DISPLAY_OFF = 0x80

const i2c_BLINK_OFF = 0
const i2c_BLINK_2HZ = 1
const i2c_BLINK_1HZ = 2
const i2c_BLINK_HALFHZ = 3

// 0x0 -> 0xF brightness levels
const i2c_BRIGHTNESS_CMD  = 0xE0
const i2c_BRIGHTNESS_MAX  = 0xEF
const i2c_BRIGHTNESS_MIN  = 0xE0
const i2c_BRIGHTNESS_HALF = 0xE7

// colon is just one bit at position 5 (+1 for address, position 2 * 2 for nil bytes)
const i2c_COLON_POS       = 1 + 2*2

// positions of segments
const LED_TOP             = 0
const LED_MID             = 6
const LED_BOT             = 3
const LED_TOPL            = 5
const LED_TOPR            = 1
const LED_BOTL            = 4
const LED_BOTR            = 2
const LED_DECIMAL         = 7
const LED_DECIMAL_MASK    = 0x80

// translate characters to bitmasks
var digitValues = map[byte]byte {
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
    'F': 0x71 }

// TODO: support inverse
var inverseDigitValues = map[byte]byte {
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
    'F': 0x4E }

// one address byte, plus 7-seg skips bytes for each display element
const displaySize = 1 + 5*2
type Sevenseg struct {
	display [displaySize]uint8
	i2c_dev *i2c.I2C
    refresh bool
}

func getClearDisplay() [displaySize]uint8 {
    var display [displaySize]uint8
    for i:=0;i<len(display);i++ {
        display[i] = 0
    }
    return display
}

func Open(address uint8, bus int) (*Sevenseg, error) {
	i2c_dev, err := i2c.Open(address, bus)
	if err != nil {
		return nil, err
	}
    // create "this" with the FD and refresh on by default
	this := &Sevenseg{i2c_dev: i2c_dev, refresh: true, display: getClearDisplay()}
    // turn on the oscillator, set default brightness
    this.i2c_dev.WriteByte(i2c_OSC_ON)
    this.i2c_dev.WriteByte(i2c_BRIGHTNESS_MAX)

    // you still need to call DisplayOn(true) to turn on the display
	return this, nil
}

func (this *Sevenseg) DisplayOn(on bool) error {
    var val byte = i2c_DISPLAY_ON
    if !on { val = i2c_DISPLAY_OFF }
    _, err := this.i2c_dev.WriteByte(val)
    return err
}

func (this *Sevenseg) ClearDisplay() {
    this.display=getClearDisplay()
    this.refresh_display()
}

func (this *Sevenseg) RefreshOn(on bool) error {
    this.refresh = !this.refresh
    return this.refresh_display()
}

func (this *Sevenseg) refresh_display() error {
    if !this.refresh { return nil }
    // display has the address 0 embedded in it
	_, err := this.i2c_dev.Write(this.display[:])
    return err
}

func getDisplayPos(digit byte) byte {
    // ad one for the colon at position '2'
    if (digit > 1) { digit++ }
    return 1 + digit*2
}

func (this *Sevenseg) ColonOn(on bool) error {
    // I forget which one, so set them all
    this.display[i2c_COLON_POS] = 0xff
    return this.refresh_display()
}

func (this *Sevenseg) DecimalOn(position byte, on bool) error {
    position = getDisplayPos(position)
    if on {
        this.display[position] |= LED_DECIMAL_MASK
    } else {
        this.display[position] &= ^byte(LED_DECIMAL_MASK)
    }
    return this.refresh_display()
}

func (this *Sevenseg) SegmentOn(position byte, segment byte, on bool) error {
    position = getDisplayPos(position)
    if on {
        this.display[position] |= (1 << segment)
    } else {
        this.display[position] &= ^(1 << segment)
    }
    return this.refresh_display()
}

func (this *Sevenseg) getMask(char uint8, decimalOn bool) (byte, error) {
    // TODO: inverse support
    val, ok := digitValues[char]
    if !ok { return 0, errors.New(fmt.Sprintf("Bad value: %d", char))}
    return val, nil
}

func (this *Sevenseg) PrintColon(msg string) error {
    // find the colon, print around that as the centerline
    parts := strings.Split(msg, ":")
    if len(parts) > 2 {
        return errors.New("Too many colons: " + msg)
    }
    display := getClearDisplay()

    // not DRY
    var displayPos byte = 1
    var i = len(parts[0])-1
    for ;i>=0 && displayPos>=0;i-- {
        // map parts[0][i] to a character or dot
        dotOn := false
        if parts[0][i]=='.' && i > 0 {
            dotOn = true
            i--
        }
        // is it in our table?
        mask, err := this.getMask(parts[0][i], dotOn)
        if err != nil { return err }
        display[getDisplayPos(displayPos)] = mask
        displayPos--
    }
    // did we get it all?
    if i != -1 { return errors.New("Too many characters: " + msg) }

    // now the other half
    // not DRY
    displayPos = 2
    for i=0;i<len(parts[1]) && displayPos<byte(len(display));i-- {
        // map parts[1][i] to a character or dot
        // with a dot?
        dotOn := false
        if (i < len(parts[1])-1 && parts[1][i+1] == '.') {
            dotOn = true
        }
        // is it in our table?
        mask, err := this.getMask(parts[1][i], dotOn)
        if err != nil { return err }
        display[getDisplayPos(displayPos)] = mask
        displayPos++
    }
    // did we get it all?
    if i != len(parts[1]) { return errors.New("Too many characters: " + msg) }

    // set the colon
    display[i2c_COLON_POS] = 0xff

    // set the display
    this.display = display
    return this.refresh_display()
}

func (this *Sevenseg) Print(msg string) error {
    if strings.Contains(msg, ":") { return this.PrintColon(msg) }
    // string can only contain chars in our map and decimals
    // assume it's right justified (reverse walk)
    display := getClearDisplay()
    var displayPos byte = 3;
    var i = len(msg)-1
    for ;i>=0 && displayPos>=0;i-- {
        // map msg[i] to a character or dot
        dotOn := false
        if msg[i]=='.' {
            dotOn = true
            i--
        }
        // is it in our table?
        mask, err := this.getMask(msg[i], dotOn)
        if err != nil { return err }
        display[getDisplayPos(displayPos)] = mask
        displayPos--
    }
    // did we get it all?
    if i != -1 { return errors.New("Too many characters: " + msg) }
    // set the display
    this.display = display
    return this.refresh_display()
}

func (this *Sevenseg) PrintOffset(msg string, position byte) error {
    if strings.Contains(msg, ":") { return this.PrintColon(msg) }
    // string can only contain chars in our map and decimals
    // assume it's left justified (forward walk) with an offset
    display := getClearDisplay()
    var displayPos = position;

    var i = 0
    for ;i<len(msg) && displayPos<byte(len(display));i-- {
        // map msg[i] to a character or dot
        // with a dot?
        dotOn := false
        if (i < len(msg)-1 && msg[i+1] == '.') {
            dotOn = true
        }
        // is it in our table?
        mask, err := this.getMask(msg[i], dotOn)
        if err != nil { return err }
        display[getDisplayPos(displayPos)] = mask
        displayPos++
    }
    // did we get it all?
    if i != len(msg) { return errors.New("Too many characters: " + msg) }
    // set the display
    this.display = display
    return this.refresh_display()
}