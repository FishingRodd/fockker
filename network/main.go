package network

import (
	"fmt"
	"fockker/network/driver"
	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"io/fs"
	nw "net"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/tabwriter"
)

func loadConfig() {
	//log.Warning("loadConfig")
	basePath := fmt.Sprintf(networkPath, "")
	// 去除地址中的%s容器名以及最后的/斜杠
	if _, err := os.Stat(basePath); err != nil {
		if os.IsNotExist(err) {
			_ = os.MkdirAll(basePath, 0644)
		} else {
			return
		}
	}
	// 读取路径下所有网络配置文件
	_ = filepath.WalkDir(basePath, func(dirPath string, d fs.DirEntry, err error) error {
		// 忽略 basePath 自身
		if dirPath == basePath {
			return nil
		}
		if d.IsDir() {
			_, networkName := path.Split(dirPath)
			// 文件夹名称即为网络名
			net := &Network{
				Name: networkName,
			}
			err = net.InfoLoad()
			if err != nil {
				log.Errorf("%s网络配置加载失败: %v", net.Name, err)
				return err
			}
			networks[net.Name] = net
			drivers[net.Driver.DriverName] = &net.Driver
		}
		return nil
	})
}

// 检查指定的桥接是否存在
func checkBridgeExists(bridgeName string) (bool, string) {
	// 获取所有网络链接（包括桥接）
	links, err := netlink.LinkList()
	if err != nil {
		return false, "" // 返回错误
	}

	// 遍历所有链接，查找指定的桥接
	for _, link := range links {
		// 检查链接类型是否为桥接，并且名称是否匹配
		if link.Type() == "bridge" && link.Attrs().Name == bridgeName {
			addrs, _ := netlink.AddrList(link, netlink.FAMILY_V4)
			return true, addrs[0].IPNet.String() // 找到网桥，返回 true和其IP地址
			// 网桥可能有多个IP地址，但本容器下不会配置多个IP，直接返回第一个IP
		}
	}

	return false, "" // 未找到桥接，返回 false
}

// InitNetwork 初始化默认网络，并加载所有已有的网络配置。
func InitNetwork() {
	// 从本地配置文件加载到实例中
	loadConfig()
	// 判断是否存在默认网络，不存在则创建
	net, fileExists := networks[DefaultBridgeName]
	hasBridgeExists, brIPRange := checkBridgeExists(DefaultBridgeName)
	// 本地配置文件不存在
	if !fileExists {
		// 默认网络不存在
		net = &Network{
			Name:        DefaultBridgeName,
			NetworkType: Bridge,
		}
		// 网桥不存在则创建
		if !hasBridgeExists {
			err := net.createNetwork(defaultSubnet)
			if err != nil {
				log.Errorf("默认网络创建失败")
				return
			}
		} else {
			net.Driver = driver.Driver{
				DriverName: net.Name,
				DriverType: driver.Bridge,
			}
			// 初始配置文件路径
			net.NetworkConfigPath = fmt.Sprintf(networkPath, net.Name)
			// 定义网络的subnet配置路径
			net.IpAllocator.SubnetAllocatorPath = path.Join(net.NetworkConfigPath, defaultAllocatorConfigName)
			_, ipRange, _ := nw.ParseCIDR(brIPRange)
			brIP := strings.Split(brIPRange, "/")[0]
			ipRange.IP = nw.ParseIP(brIP)
			net.IpRange = ipRange
			net.IpAllocator.ManualAllocate(ipRange)
		}
		err := net.InfoDump()
		if err != nil {
			log.Errorf("默认网络配置初始化失败")
			return
		}
	} else {

	}
	loadConfig()
}

// ListNetwork 列出当前所有已创建的网络及其信息。
func ListNetwork() {
	// 从本地配置文件加载到实例中
	w := tabwriter.NewWriter(os.Stdout, 12, 1, 3, ' ', 0)
	_, err := fmt.Fprint(w, "Name\tIpRange\tType\n")
	if err != nil {
		return
	}
	for _, net := range networks {
		_, err = fmt.Fprintf(w, "%s\t%s\t%s\n",
			net.Name,
			net.IpRange.String(),
			net.NetworkType,
		)
	}
	if err := w.Flush(); err != nil {
		log.Errorf("网络信息刷写异常 %v", err)
		return
	}
}

// CreateNetwork 创建网络
func CreateNetwork(networkName string, networkType NetworkType, subnet string) error {
	// 未指定网段则使用默认
	if subnet == "" {
		subnet = defaultSubnet
	}
	// TODO 网段冲突判断
	net := &Network{
		Name:        networkName,
		NetworkType: networkType,
	}
	switch net.NetworkType {
	case Bridge:
		// 创建网络
		err := net.createNetwork(subnet)
		if err != nil {
			return fmt.Errorf("%s网络创建失败: %v", networkName, err)
		}
		// 写入网络配置信息
		err = net.InfoDump()
		if err != nil {
			return fmt.Errorf("%s网络配置写入失败: %v", networkName, err)
		}
	case Host:
		log.Infof("TODO")
	case None:
		log.Infof("TODO")
	default:
		return fmt.Errorf("不支持的网络类型 %s", networkType)
	}
	fmt.Printf("网络: %s, 创建成功\n", networkName)
	return nil
}

// ConnectToNetwork 连接容器到网络
func ConnectToNetwork(networkName string, containerID string, containerPortMapping []string, containerPID string) {
	net, exists := networks[networkName]
	if !exists {
		log.Errorf("连接失败，网络%s 不存在", networkName)
		return
	}
	err := net.connect(containerID, containerPortMapping, containerPID)
	if err != nil {
		log.Errorf("%s网络连接失败", networkName)
		return
	}
}

// DisconnectFromNetwork 容器断开网络
func DisconnectFromNetwork(networkName string, containerID string) {
	net, exists := networks[networkName]
	if !exists {
		log.Errorf("网络%s 不存在", networkName)
		return
	}
	err := net.disconnect(containerID)
	if err != nil {
		log.Errorf("%s网络断开失败: %v", networkName, err)
		return
	}
}

// DistoryNetwork 删除网络
func DistoryNetwork(networkName string) error {
	net, exists := networks[networkName]
	if !exists {
		return fmt.Errorf("删除失败，网络%s 不存在", networkName)
	}
	err := net.deleteNetwork()
	if err != nil {
		return fmt.Errorf("网络删除异常: %v", err)
	}
	fmt.Printf("网络: %s, 删除成功\n", networkName)
	return nil
}
