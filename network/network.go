package network

import (
	"encoding/json"
	"fmt"
	"fockker/network/driver"
	"fockker/network/iptables"
	log "github.com/sirupsen/logrus"
	nw "net"
	"os"
	"path"
	"strings"
)

// InfoDump 将当前网络的配置（包括名称、IP范围和驱动）以JSON格式保存到配置路径。
func (net *Network) InfoDump() error {
	net.NetworkConfigPath = fmt.Sprintf(networkPath, net.Name)
	// 创建网络配置目录
	if _, err := os.Stat(net.NetworkConfigPath); err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(net.NetworkConfigPath, 0644)
			if err != nil {
				log.Errorf("%s网络配置 目录创建异常: %v", net.NetworkConfigPath, err)
				return err
			}
		} else {
			return err
		}
	}
	// 写入网络配置信息
	networkFilePath := path.Join(net.NetworkConfigPath, defaultNetworkConfigName)
	configFile, err := os.OpenFile(networkFilePath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Errorf("%s网络配置 文件打开异常: %v", networkFilePath, err)
		return err
	}
	defer func(configFile *os.File) {
		err := configFile.Close()
		if err != nil {

		}
	}(configFile)

	netJson, err := json.Marshal(net)
	if err != nil {
		log.Errorf("网络配置序列化异常: %v", err)
		return err
	}

	_, err = configFile.Write(netJson)
	if err != nil {
		log.Errorf("网络配置写入异常: %v", err)
		return err
	}
	return nil
}

// InfoLoad 从指定路径加载网络配置文件，并解析JSON数据填充当前网络实例。
func (net *Network) InfoLoad() error {
	net.NetworkConfigPath = fmt.Sprintf(networkPath, net.Name)
	net.IpAllocator.SubnetAllocatorPath = path.Join(net.NetworkConfigPath, defaultAllocatorConfigName)
	networkFilePath := path.Join(net.NetworkConfigPath, defaultNetworkConfigName)

	configFile, err := os.Open(networkFilePath)
	defer func(configFile *os.File) {
		err := configFile.Close()
		if err != nil {

		}
	}(configFile)

	if err != nil {
		log.Errorf("%s网络配置 文件打开异常: %v", net.Name, err)
		return err
	}
	netJson := make([]byte, 2000)
	n, err := configFile.Read(netJson)
	if err != nil {
		log.Errorf("网络配置读取异常: %v", err)
		return err
	}

	err = json.Unmarshal(netJson[:n], net)
	if err != nil {
		log.Errorf("网络配置反序列化异常: %v", err)
		return err
	}
	return nil
}

// 创建新的网络并将其信息保存到配置路径。
func (net *Network) createNetwork(subnet string) error {
	// 初始配置文件路径
	net.NetworkConfigPath = fmt.Sprintf(networkPath, net.Name)
	// 定义网络的subnet配置路径
	net.IpAllocator.SubnetAllocatorPath = path.Join(net.NetworkConfigPath, defaultAllocatorConfigName)

	net.Driver = driver.Driver{DriverName: net.Name}
	// 判断当前driver是否已存在
	if _, exists := drivers[net.Driver.DriverName]; exists {
		log.Errorf("驱动%s 已存在, 无需创建", net.Driver.DriverName)
	} else {
		switch net.NetworkType {
		case Bridge:
			net.Driver.DriverType = driver.Bridge
			// 网段分配
			_, ipRange, _ := nw.ParseCIDR(subnet)
			// IP分配
			gatewayIP, err := net.IpAllocator.Allocate(ipRange)
			if err != nil {
				return err
			}
			ipRange.IP = gatewayIP
			net.IpRange = ipRange
			// 创建驱动
			err = net.Driver.Create(net.IpRange)
			if err != nil {
				return err
			}
			// 将创建好的driver加入到drivers
			drivers[net.Driver.DriverName] = &net.Driver
		case Host:
			return nil
		}
	}

	return nil
}

// 删除指定的网络、驱动和分配的IP。
func (net *Network) deleteNetwork() error {
	if net.NetworkType != None {
		if err := net.Driver.Delete(); err != nil {
			return fmt.Errorf("删除网络驱动时异常: %v", err)
		}
	}
	if err := os.RemoveAll(net.NetworkConfigPath); err != nil {
		return fmt.Errorf("删除网络配置路径%s 异常: %v", net.NetworkConfigPath, err)
	}
	return nil
}

// 连接容器到指定网络
func (net *Network) connect(containerID string, containerPortMapping []string, containerPID string) error {
	// 分配容器IP地址
	ip, err := net.IpAllocator.Allocate(net.IpRange)
	if err != nil {
		log.Errorf("%v", err)
		return err
	}
	// 创建网络端点
	endpointId := fmt.Sprintf("%s-%s", containerID, net.Name)
	ep := &Endpoint{
		ID:          endpointId,
		IPAddress:   ip,
		Network:     net,
		PortMapping: containerPortMapping,
	}
	// 调用网络驱动挂载和配置网络端点
	if err = net.Driver.ConnectBridge(ep.ID[:5], &ep.Device); err != nil {
		log.Errorf("网桥连接异常 %v", err)
		return err
	}
	// 进入容器namespace配置容器网络设备IP地址
	if err = configEndpointIpAddressAndRoute(ep, containerPID); err != nil {
		log.Errorf("容器接口异常 %v", err)
		return err
	}
	return net.configPortMapping(ep)
}

// 断开容器与网络的连接
func (net *Network) disconnect(containerID string) error {
	endpointId := fmt.Sprintf("%s-%s", containerID, net.Name)
	err := net.Driver.DisconnectBridge(endpointId[:5])
	if err != nil {
		return err
	}
	return nil
}

// 配置宿主机到容器的端口映射
func (net *Network) configPortMapping(ep *Endpoint) error {
	for _, pm := range ep.PortMapping {
		portMapping := strings.Split(pm, ":")
		if len(portMapping) != 2 {
			log.Errorf("端口映射格式错误 %v", pm)
			continue
		}
		err := iptables.OuterToInner(portMapping[0], ep.IPAddress.String(), portMapping[1])
		if err != nil {
			continue
		}
	}
	return nil
}

// 判断两个网络是否属于同一网络位
func isSameNetwork(net1, net2 *nw.IPNet) bool {
	// 比较掩码和网络地址
	return net1.IP.Equal(net2.IP) && net1.Mask.String() == net2.Mask.String()
}

// 生成新网络配置
func incrementNetwork(ipNet *nw.IPNet) string {
	// 获取掩码位数
	ones, _ := ipNet.Mask.Size()

	// 计算网络块增量
	blockSize := 1 << (32 - ones)

	// 将IP转换为32位整型
	ipInt := ipToInt(ipNet.IP)

	// 计算新网络地址
	newIP := intToIP(ipInt + uint32(blockSize))

	// 返回新CIDR格式
	return fmt.Sprintf("%s/%d", newIP, ones)
}

// IP转32位整型
func ipToInt(ip nw.IP) uint32 {
	ip = ip.To4()
	return uint32(ip[0])<<24 | uint32(ip[1])<<16 | uint32(ip[2])<<8 | uint32(ip[3])
}

// 32位整型转IP
func intToIP(n uint32) nw.IP {
	return nw.IPv4(
		byte(n>>24),
		byte(n>>16),
		byte(n>>8),
		byte(n),
	)
}
