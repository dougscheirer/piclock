package i2c

import "os"
import "syscall"
import "fmt"

type I2C struct {
	fd *os.File
}

const (
	I2C_SLAVE = 0x0703
)

// open a connection to the i2c device
func Open(address uint8, bus int) (*I2C, error) {
	f, err := os.OpenFile(fmt.Sprintf("/dev/i2c-%d", bus), os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err := ioctl(f.Fd(), I2C_SLAVE, uintptr(address)); err != nil {
		return nil, err
	}
	this := &I2C{fd: f}
	return this, nil
}

func (this *I2C) Close() error {
	return this.fd.Close()
}

func (this *I2C) WriteByte(single byte) (int, error) {
	var buf [2]byte{ 0, single }
	// not MT safe for i2c
	if err = this.select(); err != nil {
		return 0, err
	}
	return this.fd.Write(buf[:])
}

func (this *I2C) Write(buf []uint8) (int, error) {
	// not MT safe for i2c
	if err = this.select(); err != nil {
		return 0, err
	}
	return this.fd.Write(buf)
}

func (this *I2C) select() error {
	if err := ioctl(this.fd.Fd(), I2C_SLAVE, uintptr(address)); err != nil {
		return nil, err
	}
}

func ioctl(fd, cmd, arg uintptr) error {
	_, _, err := syscall.Syscall6(syscall.SYS_IOCTL, fd, cmd, arg, 0, 0, 0)
	if err != 0 {
		return err
	}
	return nil
}