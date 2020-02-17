package sevenseg_backpack

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"dscheirer.com/piclock/i2c"
)

// commands we support
// OSC on/off 0/1
const i2c_OSC_CMD = 0x20
const i2c_OSC_ON = 0x21
const i2c_OSC_OFF = 0x20

// display on/off and 2 "blink" bits in position 2+1
const i2cDISPLAY_CMD = 0x80
const i2cDISPLAY_ON = 0x81
const i2cDISPLAY_OFF = 0x80

// 0x0 -> 0xF brightness levels
const i2cBRIGHTNESS_CMD = 0xE0
const i2cBRIGHTNESS_MAX = 0xEF
const i2cBRIGHTNESS_MIN = 0xE0
const i2cBRIGHTNESS_HALF = 0xE7

// colon is just one bit at position 5 (+1 for address, position 2 * 2 for nil bytes)
const i2c_COLON_POS = 1 + 2*2

// export blink positions
const BLINK_OFF = 0
const BLINK_2HZ = 1
const BLINK_1HZ = 2
const BLINK_HALFHZ = 3

// positions of segments
const LED_TOP = 0
const LED_MID = 6
const LED_BOT = 3
const LED_TOPL = 5
const LED_TOPR = 1
const LED_BOTL = 4
const LED_BOTR = 2
const LED_DECIMAL = 7
const LED_DECIMAL_MASK = 0x80

// translate characters to bitmasks
var digitValues = map[byte]byte{
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
	'c': 0x58,
	'D': 0x5E,
	'E': 0x79,
	'F': 0x71,
	'R': 0x50,
	'H': 0x76,
	'h': 0x74,
	'l': 0x06,
	'L': 0x38,
	'n': 0x54,
	'o': 0x5C,
	'O': 0x3F,
	'S': 0x6D,
	'I': 0x06,
	'P': 0x73,
	'u': 0x1C,
	'i': 0x04,
	'U': 0x3E,
	't': 0x78,
	'Y': 0x6E,
}

// TODO: support inverse
var inverseDigitValues = map[byte]byte{
	' ': 0x00,
	'-': 0x40,
	'_': 0x08, // TODO: support these
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
	'c': 0x43,
	'D': 0x73,
	'E': 0x4F,
	'F': 0x4E,
	'R': 0x42,
	'H': 0x76,
	'h': 0x66,
	'l': 0x30,
	'L': 0x70,
	'n': 0x52,
	'o': 0x56,
	'O': 0x3F,
	'S': 0x6D,
	'I': 0x06,
	'i': 0x04,
	'P': 0x73,
	'u': 0x1C,
	'U': 0x3E,
	't': 0x78,
	'Y': 0x6E,
}

// one address byte, plus 7-seg skips bytes for each display element
const displaySize = 1 + 5*2

type Sevenseg struct {
	display        [displaySize]uint8
	i2cDev         *i2c.I2C
	refresh        bool
	inverted       bool
	dump           bool
	blink          byte
	sim            bool
	currentDisplay [displaySize]uint8
}

func (this *Sevenseg) simLog(v string, args ...interface{}) {
	if !this.sim {
		return
	}
	log.Printf(v, args...)
}

func getClearDisplay() [displaySize]uint8 {
	var display [displaySize]uint8
	for i := 0; i < len(display); i++ {
		display[i] = 0
	}
	return display
}

func Open(address uint8, bus int, simulated bool) (*Sevenseg, error) {
	i2cDev, err := i2c.Open(address, bus, simulated)
	if err != nil {
		return nil, err
	}
	// create "this" with the FD and refresh on by default
	this := &Sevenseg{
		i2cDev:         i2cDev,
		refresh:        true,
		inverted:       false,
		blink:          BLINK_OFF,
		dump:           false,
		display:        getClearDisplay(),
		sim:            simulated,
		currentDisplay: getClearDisplay()}
	// turn on the oscillator, set default brightness
	this.i2cDev.WriteByte(i2c_OSC_ON)
	this.i2cDev.WriteByte(i2cBRIGHTNESS_MAX)
	// you still need to call DisplayOn(true) to turn on the display
	return this, nil
}

func (this *Sevenseg) DebugDump(on bool) {
	this.dump = on
}

func (this *Sevenseg) SetInverted(inverted bool) {
	this.simLog("Inverted: %t", inverted)
	this.inverted = inverted
}

func (this *Sevenseg) DisplayOn(on bool) error {
	this.simLog("Display: %t", on)
	// blink rate is bits 2 and 1 of the display command
	var val byte = i2cDISPLAY_ON | (this.blink << 1)
	if !on {
		val = i2cDISPLAY_OFF
	}
	_, err := this.i2cDev.WriteByte(val)
	return err
}

func (this *Sevenseg) ClearDisplay() {
	this.simLog("ClearDisplay")
	this.display = getClearDisplay()
	this.refresh_display()
}

func (this *Sevenseg) RefreshOn(on bool) error {
	this.simLog("Refresh: %t", on)
	this.refresh = !this.refresh
	return this.refresh_display()
}

func (this *Sevenseg) dumpDisplay() {
	//  -     -      -     -
	// | |   | |  . | |   | |
	//  -     -      -     -
	// | |   | |  . | |   | |
	//  -  .  -  .   -  .  -  .

	// go one row at a time
	// TOP
	var i byte
	line := "\n"
	for i = 0; i < 4; i++ {
		if i == 2 {
			line += " "
		}
		if this.display[getDisplayPos(i)]&(1<<LED_TOP) != 0 {
			line += "  -   "
		} else {
			line += "      "
		}
	}
	// TOPM
	line += "\n"
	for i = 0; i < 4; i++ {
		if i == 2 {
			if this.display[i2c_COLON_POS] != 0 {
				line += "."
			} else {
				line += " "
			}
		}
		if this.display[getDisplayPos(i)]&(1<<LED_TOPL) != 0 {
			line += " |"
		} else {
			line += "  "
		}
		if this.display[getDisplayPos(i)]&(1<<LED_TOPR) != 0 {
			line += " |  "
		} else {
			line += "    "
		}
	}
	// MID
	line += "\n"
	for i = 0; i < 4; i++ {
		if i == 2 {
			line += " "
		}
		if this.display[getDisplayPos(i)]&(1<<LED_MID) != 0 {
			line += "  -   "
		} else {
			line += "      "
		}
	}
	// BOTM
	line += "\n"
	for i = 0; i < 4; i++ {
		if i == 2 {
			if this.display[i2c_COLON_POS] != 0 {
				line += "."
			} else {
				line += " "
			}
		}
		if this.display[getDisplayPos(i)]&(1<<LED_BOTL) != 0 {
			line += " |"
		} else {
			line += "  "
		}
		if this.display[getDisplayPos(i)]&(1<<LED_BOTR) != 0 {
			line += " |  "
		} else {
			line += "    "
		}
	}
	// BOT
	line += "\n"
	for i = 0; i < 4; i++ {
		if i == 2 {
			line += " "
		}
		if this.display[getDisplayPos(i)]&(1<<LED_BOT) != 0 {
			line += "  -  "
		} else {
			line += "     "
		}
		if this.display[getDisplayPos(i)]&(1<<LED_DECIMAL) != 0 {
			line += "."
		} else {
			line += " "
		}
	}
	line += "\n"
	log.Println(line)
}

func (this *Sevenseg) refresh_display() error {
	if !this.refresh {
		return nil
	}
	// refreshing on the same thing?
	if this.currentDisplay == this.display {
		return nil
	}
	// set the display buffer
	this.currentDisplay = this.display

	// display has the address 0 embedded in it
	// for debugging, dump out what we think we're putting on the display
	if this.dump {
		this.dumpDisplay()
	}

	_, err := this.i2cDev.Write(this.display[:])
	return err
}

func getDisplayPos(digit byte) byte {
	// ad one for the colon at position '2'
	if digit > 1 {
		digit++
	}
	return 1 + digit*2
}

func (this *Sevenseg) ColonOn(on bool) error {
	this.simLog("Colon: %t", on)
	// I forget which one, so set them all
	this.display[i2c_COLON_POS] = 0xff
	return this.refresh_display()
}

func (this *Sevenseg) DecimalOn(position byte, on bool) error {
	this.simLog("Decimal %d: %t", position, on)
	position = getDisplayPos(position)
	if on {
		this.display[position] |= LED_DECIMAL_MASK
	} else {
		this.display[position] &= ^byte(LED_DECIMAL_MASK)
	}
	return this.refresh_display()
}

func (this *Sevenseg) SegmentOn(position byte, segment byte, on bool) error {
	this.simLog("Segment %d/%d: %t", position, segment, on)
	position = getDisplayPos(position)
	if on {
		this.display[position] |= (1 << segment)
	} else {
		this.display[position] &= ^(1 << segment)
	}
	return this.refresh_display()
}

func altCase(char uint8) uint8 {
	if char >= 'A' && char <= 'Z' {
		return char + 'a' - 'A'
	} else if char >= 'a' && char <= 'z' {
		return char + 'A' - 'a'
	}
	return char
}
func (this *Sevenseg) getMask(char uint8, decimalOn bool) (byte, error) {
	if char == '.' {
		// pretend it's a space
		char = ' '
	}

	// TODO: inverse support
	var val uint8
	var ok bool
	if !this.inverted {
		val, ok = digitValues[char]
	} else {
		val, ok = inverseDigitValues[char]
	}
	if !ok {
		// try alternate cases
		if !this.inverted {
			val, ok = digitValues[altCase(char)]
		} else {
			val, ok = inverseDigitValues[altCase(char)]
		}
		if !ok {
			return 0, errors.New(fmt.Sprintf("Bad value: %s", string(char)))
		}
	}
	if decimalOn {
		val |= (1 << LED_DECIMAL)
	}
	return val, nil
}

func (this *Sevenseg) PrintColon(msg string) error {
	// find the colon, print around that as the centerline
	parts := strings.Split(msg, ":")
	if len(parts) > 2 {
		return errors.New("Too many colons: " + msg)
	}
	display := getClearDisplay()
	// first do parts[0]
	// not DRY
	var displayPos = 1
	var inc = -1
	var bound = -1
	if this.inverted {
		displayPos = 2
		inc = +1
		bound = 4
	}
	var i = len(parts[0]) - 1
	for ; i >= 0 && displayPos != bound; i-- {
		// map parts[0][i] to a character or dot
		dotOn := false
		if parts[0][i] == '.' && i > 0 {
			dotOn = true
			i--
		}
		// is it in our table?
		mask, err := this.getMask(parts[0][i], dotOn)
		if err != nil {
			return err
		}
		display[getDisplayPos(byte(displayPos))] = mask
		displayPos += inc
	}
	// did we get it all?
	if i != -1 {
		return errors.New("Too many characters: " + msg)
	}

	// now the other half
	// not DRY
	displayPos = 2
	inc = +1
	bound = 4
	if this.inverted {
		displayPos = 1
		inc = -1
		bound = -1
	}
	for i = 0; i < len(parts[1]) && displayPos != bound; i++ {
		// map parts[1][i] to a character or dot
		// with a dot?
		dotOn := false
		if i < len(parts[1])-1 && parts[1][i+1] == '.' {
			dotOn = true
		}
		// is it in our table?
		mask, err := this.getMask(parts[1][i], dotOn)
		if err != nil {
			return err
		}
		display[getDisplayPos(byte(displayPos))] = mask
		displayPos += inc
		if dotOn {
			i++
		}
	}
	// did we get it all?
	if i != len(parts[1]) {
		return errors.New("Too many characters: " + msg)
	}

	// set the colon
	display[i2c_COLON_POS] = 0xff

	// set the display
	this.display = display
	return this.refresh_display()
}

func (this *Sevenseg) PrintFromPosition(msg string, position int) error {
	if strings.Contains(msg, ":") {
		return this.PrintColon(msg)
	}
	// string can only contain chars in our map and decimals
	// assume it's left justified (forward walk) with an offset
	display := getClearDisplay()
	var displayPos = position
	var inc = +1
	var bound = 4
	if this.inverted {
		displayPos = 3 - position
		inc = -1
		bound = -1
	}

	var i = 0
	for ; i < len(msg) && displayPos != bound; i++ {
		// map msg[i] to a character or dot
		// with a dot?
		dotOn := false
		if i < len(msg)-1 && msg[i+1] == '.' {
			dotOn = true
		}
		// is it in our table?
		mask, err := this.getMask(msg[i], dotOn)
		if err != nil {
			return err
		}
		display[getDisplayPos(byte(displayPos))] = mask
		displayPos += inc
	}
	// did we get it all?
	if i != len(msg) {
		return errors.New("Too many characters: " + msg)
	}
	// set the display
	this.display = display
	return this.refresh_display()
}

// Given a string and a start point, print as much as you can (left -> right)
func (this *Sevenseg) PrintOffset(msg string, offset int) (string, error) {
	// TODO: adjust for inverse using displayPos and direction
	display := getClearDisplay()
	var displayPos = 0
	var inc = +1
	var bound = 4
	if this.inverted {
		displayPos = 0
		inc = -1
		bound = -1
	}
	var i int
	for i = offset; i < len(msg) && displayPos != bound; i++ {
		// map msg[i] to a character
		target := msg[i]
		dotOn := false
		if target == '.' {
			dotOn = true
			target = ' '
		} else if i+1 < len(msg) && msg[i+1] == '.' {
			dotOn = true
			i++
		}
		// is it in our table?
		mask, err := this.getMask(target, dotOn)
		if err != nil {
			return "", err
		}
		display[getDisplayPos(byte(displayPos))] = mask
		displayPos += inc
	}
	// set the display
	this.display = display
	return msg[offset:i], this.refresh_display()
}

func (this *Sevenseg) Print(msg string) error {
	if strings.Contains(msg, ":") {
		return this.PrintColon(msg)
	}
	// string can only contain chars in our map and decimals
	// assume it's right justified (reverse walk)
	// TODO: adjust for inverse using displayPos and direction
	display := getClearDisplay()
	var displayPos = 3
	var inc = -1
	var bound = -1
	if this.inverted {
		displayPos = 0
		inc = +1
		bound = 4
	}
	var i = len(msg) - 1
	for ; i >= 0 && displayPos != bound; i-- {
		// map msg[i] to a character or dot
		target := msg[i]
		dotOn := false
		if msg[i] == '.' {
			dotOn = true
			// if the char before this is also a '.', don't skip
			// assume that there is a space between them
			i--
			if i >= 0 {
				if msg[i] == '.' {
					target = ' '
				} else {
					target = msg[i]
				}
			} else {
				target = ' '
				i = 0
			}
		}
		// is it in our table?
		mask, err := this.getMask(target, dotOn)
		if err != nil {
			return err
		}
		display[getDisplayPos(byte(displayPos))] = mask
		displayPos += inc
	}
	// did we get it all?
	if i != -1 {
		return errors.New("Too many characters: " + msg)
	}
	// set the display
	this.display = display
	return this.refresh_display()
}

func (this *Sevenseg) SetBlinkRate(rate uint8) error {
	if rate > 3 {
		return errors.New(fmt.Sprintf("Bad blink rate: %d", rate))
	}
	this.simLog("Blink rate %d", rate)
	this.blink = rate
	// one assumes you want the display on now?
	return this.DisplayOn(true)
}

func (this *Sevenseg) SetBrightness(level uint8) error {
	if level > 15 {
		return errors.New(fmt.Sprintf("Bad brightness level: %d", level))
	}
	this.simLog("Brightness %d", level)
	_, err := this.i2cDev.WriteByte(i2cBRIGHTNESS_CMD | level)
	return err
}
