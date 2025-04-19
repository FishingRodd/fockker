package network

import (
	"fockker/constants"
	"fockker/network/driver"
	"fockker/network/ipam"
	"github.com/vishvananda/netlink"
	"net"
)

type NetworkType string

const (
	Bridge NetworkType = "bridge"
	Host   NetworkType = "host"
	None   NetworkType = "none"
)

const (
	DefaultBridgeName          string = "fockker0"                        // 默认的Bridge类型驱动名
	defaultSubnet              string = "192.168.0.0/24"                  // 默认网段
	defaultNetworkConfigName   string = "config.json"                     // 网络配置文件名称
	defaultAllocatorConfigName string = "subnet.json"                     // IP分配文件名称
	networkPath                string = constants.RunPath + "/network/%s" // 网络配置存储路径，%s为网络名
)

var (
	networks = map[string]*Network{}       // 网络名:{}
	drivers  = map[string]*driver.Driver{} // 驱动名:{}
)

type Network struct {
	Name              string        `json:"NetworkName"` // 网络名称
	IpRange           *net.IPNet    `json:"IpRange"`     // 网络的IP范围
	Driver            driver.Driver `json:"Driver"`      // 网络驱动
	NetworkType       NetworkType   `json:"NetworkType"` // 网络类型
	IpAllocator       ipam.IPAM     `json:"-"`           // 每个网络都存在IP分配器
	NetworkConfigPath string        `json:"-"`           // 由networkPath和networkname拼接
}

type Endpoint struct {
	ID        string       // 端点的唯一标识
	Device    netlink.Veth // veth设备
	IPAddress net.IP       // 端点的IP地址
	//MacAddress  net.HardwareAddr // 端点的MAC地址
	Network     *Network // 端点所属的网络
	PortMapping []string // 端口映射配置
}
