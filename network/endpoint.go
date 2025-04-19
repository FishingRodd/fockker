package network

import (
	"fmt"
	"fockker/network/driver"
	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
	"net"
	"os"
	"runtime"
)

// 配置端点的IP地址和路由信息
func configEndpointIpAddressAndRoute(ep *Endpoint, containerPid string) error {
	// 根据端点的 PeerName 获取对应的网络接口
	peerLink, err := netlink.LinkByName(ep.Device.PeerName)
	if err != nil {
		return fmt.Errorf("端点配置异常: %v", err)
	}
	// 在函数结束时切换回原来的网络命名空间
	defer enterContainerNetns(&peerLink, containerPid)()
	// 获取并设置接口的 IP 地址
	interfaceIP := *ep.Network.IpRange // 获取网络范围对象
	interfaceIP.IP = ep.IPAddress      // 设置接口IP为端点地址
	// 设置接口IP
	if err = driver.SetInterfaceIP(ep.Device.PeerName, interfaceIP.String()); err != nil {
		return fmt.Errorf("%v IP接入异常: %s", ep.Network, err)
	}
	// 启动接口
	if err = driver.SetInterfaceUP(ep.Device.PeerName); err != nil {
		return fmt.Errorf("启动%s接口时异常 %v", ep.Device.PeerName, err)
	}
	// 将回环接口 (lo) 设置为 UP 状态，确保主机可以自我通信
	if err = driver.SetInterfaceUP("lo"); err != nil {
		return fmt.Errorf("启动%s接口时异常 %v", "lo", err)
	}
	// 创建一个默认路由，指向 0.0.0.0/0（代表所有地址）
	_, cidr, _ := net.ParseCIDR("0.0.0.0/0") // 解析 CIDR 地址
	// 创建一个路由对象，指定网关和目标地址
	defaultRoute := &netlink.Route{
		LinkIndex: peerLink.Attrs().Index, // 关联的接口索引
		Gw:        ep.Network.IpRange.IP,  // 网关地址，使用网络范围的 IP
		Dst:       cidr,                   // 默认路由目的地
	}
	// 添加默认路由到路由表
	if err = netlink.RouteAdd(defaultRoute); err != nil {
		return err
	}
	return nil
}

func enterContainerNetns(enLink *netlink.Link, containerPid string) func() {
	// 根据进程ID获取net namespace
	f, err := os.OpenFile(fmt.Sprintf("/proc/%s/ns/net", containerPid), os.O_RDONLY, 0)
	if err != nil {
		log.Errorf("error get container net namespace, %v", err)
	}

	nsFD := f.Fd()
	// 锁定当前线程到OS线程
	// 由于网络命名空间的切换是与线程绑定的，确保当前 Go 协程在切换网络命名空间后不会被操作系统调度到其他线程
	runtime.LockOSThread()

	// 设置 虚拟以太网接口的netns 为目标容器的netns
	if err = netlink.LinkSetNsFd(*enLink, int(nsFD)); err != nil {
		log.Errorf("error set link netns , %v", err)
	}

	// 获取当前的网络namespace，用于退出函数后恢复
	origns, err := netns.Get()
	if err != nil {
		log.Errorf("error get current netns, %v", err)
	}

	// 切换当前进程的网络命名空间，使得当前进程在目标容器的网络环境中运行
	if err = netns.Set(netns.NsHandle(nsFD)); err != nil {
		log.Errorf("error set netns, %v", err)
	}
	return func() {
		// 恢复到原来的网络命名空间
		err := netns.Set(origns)
		if err != nil {
			log.Errorf("%v", err)
		}
		// 关闭原网络命名空间句柄
		err = origns.Close()
		if err != nil {
			log.Errorf("%v", err)
		}
		// 解锁线程
		runtime.UnlockOSThread()
		// 关闭文件句柄
		err = f.Close()
		if err != nil {
			log.Errorf("%v", err)
		}
	}
}
