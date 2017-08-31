package i2c

import "os"
import "syscall"
import "fmt"

type I2C struct {
	fd 			*os.File
	address uint8
	fd_sim	bool
}

const (
	I2C_SLAVE = 0x0703
)

func logWrite(buf []uint8) error {
	fmt.Printf("Write : ")
	for i:=0;i<len(buf);i++ {
		fmt.Printf("%02x ", buf[i])
	}
	fmt.Printf("\n")
	return nil
}

func logMsg(msg string) error {
	fmt.Println(msg)
	return nil
}

// open a connection to the i2c device
func Open(address uint8, bus int, simulated bool) (*I2C, error) {
	if !simulated {
		f, err := os.OpenFile(fmt.Sprintf("/dev/i2c-%d", bus), os.O_RDWR, 0600)
		if err != nil {
			return nil, err
		}
		if err := ioctl(f.Fd(), I2C_SLAVE, uintptr(address)); err != nil {
			return nil, err
		}
		this := &I2C{fd: f, address: address, fd_sim: false}
		return this, nil
	} else {
		this := &I2C{fd_sim: true, address: address, fd: nil}
		return this, nil
	}
}

func (this *I2C) Close() error {
	if this.fd_sim { return logMsg(fmt.Sprintf("Close: %d", this.address)) }
	return this.fd.Close()
}

// this is to write a command-style byte
func (this *I2C) WriteByte(single byte) (int, error) {
	var buf [1]byte;
	buf[0] = single
	// not MT safe for i2c
	if err := select_line(this); err != nil {
		return 0, err
	}

	if this.fd_sim {
		return 0, logWrite(buf[:])
	} else {
		return this.fd.Write(buf[:])
	}
}

func (this *I2C) Write(buf []uint8) (int, error) {
	// not MT safe for i2c
	if err := select_line(this); err != nil {
		return 0, err
	}
	if this.fd_sim {
		return 0, logWrite(buf)
	} else {
		return this.fd.Write(buf)
	}
}

func select_line(this *I2C) error {
	if this.fd_sim {
		return logMsg(fmt.Sprintf("ioctl: I2C_SLAVE @ 0x%02x", this.address))
	} else {
		return ioctl(this.fd.Fd(), I2C_SLAVE, uintptr(this.address))
	}
}

func ioctl(fd, cmd, arg uintptr) error {
	_, _, err := syscall.Syscall6(syscall.SYS_IOCTL, fd, cmd, arg, 0, 0, 0)
	if err != 0 {
		return err
	}
	return nil
}
