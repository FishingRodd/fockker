package driver

import (
	"github.com/vishvananda/netlink"
	"net"
)

func (driver *Driver) Create(ipRange *net.IPNet) error {
	switch driver.DriverType {
	case Bridge:
		err := driver.createBridge(ipRange)
		return err
	}
	return nil
}

func (driver *Driver) Delete() error {
	br, err := netlink.LinkByName(driver.DriverName)
	if err != nil {
		return err
	}
	return netlink.LinkDel(br)
}
