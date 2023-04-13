package params

import "syscall"

type IOThrottle struct {
	ReadIOPS  uint64
	WriteIOPS uint64
	ReadBps   uint64
	WriteBps  uint64
}

type DeviceNumber struct {
	Major uint64
	Minor uint64
}

// GetDeviceNumber will return linux device number it will return major and minor device number
func GetDeviceNumber(deviceName string) (*DeviceNumber, error) {
	stat := syscall.Stat_t{}
	if err := syscall.Stat(deviceName, &stat); err != nil {
		return nil, err
	}
	return &DeviceNumber{
		Major: uint64(stat.Rdev / 256),
		Minor: uint64(stat.Rdev % 256),
	}, nil
}
