package driver

import (
	"fmt"
	"fockker/network/iptables"
	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"net"
)

// 分配Bridge网络，设置interface
func (bridge *Driver) createBridge(ipRange *net.IPNet) error {
	// 创建bridge接口
	if err := CreateBridgeInterface(bridge.DriverName); err != nil {
		return fmt.Errorf("添加 %s 桥接异常: %v", bridge.DriverName, err)
	}
	// 设置接口IP
	if err := SetInterfaceIP(bridge.DriverName, ipRange.String()); err != nil {
		return fmt.Errorf("%s 桥接IP %s 异常:%v", bridge.DriverName, ipRange.IP, err)
	}
	// 启动接口
	if err := SetInterfaceUP(bridge.DriverName); err != nil {
		return fmt.Errorf("启动 %s 桥接接口时异常 %v", bridge.DriverName, err)
	}
	// 设置iptables，保证容器和宿主机外部的网络通信
	if err := iptables.InnerToOuter(bridge.DriverName, ipRange); err != nil {
		return fmt.Errorf("%s 设置 iptables 时异常 %v", bridge.DriverName, err)
	}
	return nil
}

// ConnectBridge 通过veth设备对连接到网桥
func (bridge *Driver) ConnectBridge(linkID string, device *netlink.Veth) error {
	br, err := netlink.LinkByName(bridge.DriverName) // 获取指定名称的链路
	if err != nil {
		return err
	}
	// 创建新的链路属性
	la := netlink.NewLinkAttrs()
	la.Name = linkID                  // 设置链路名称
	la.MasterIndex = br.Attrs().Index // 设置主链路的索引
	// 设置虚拟以太网接口
	device.LinkAttrs = la
	device.PeerName = "cif-" + linkID // 设置对端名称

	if err = netlink.LinkAdd(device); err != nil { // 添加链路
		log.Errorf("端点添加异常: %v", err)
		return err
	}

	if err = netlink.LinkSetUp(device); err != nil { // 设置链路为UP状态
		log.Errorf("端点状态设置异常: %v", err)
		return err
	}
	return nil
}

// DisconnectBridge 通过veth设备对断开网桥
func (bridge *Driver) DisconnectBridge(linkID string) error {
	// 获取指定名称的虚拟以太网接口
	device, err := netlink.LinkByName(linkID)
	if err != nil {
		return fmt.Errorf("无法找到设备 %s: %v", linkID, err)
	}

	// 删除一端接口，linux会同时删除对端设备
	if err := netlink.LinkDel(device); err != nil {
		return fmt.Errorf("删除设备 %s 异常: %v", linkID, err)
	}
	return nil
}
