package ipam

import (
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net"
	"os"
	"path"
	"strings"
)

// 这里基于原作者的bitmap位移法代码有个致命问题，在操作如172.16.0.0/24的subnet时，会由于位移的主机位过多而导致Error dump allocation info, unexpected end of JSON input

func (ipam *IPAM) load() error {
	if _, err := os.Stat(ipam.SubnetAllocatorPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		} else {
			return err
		}
	}
	subnetConfigFile, err := os.Open(ipam.SubnetAllocatorPath)
	defer subnetConfigFile.Close()
	if err != nil {
		return err
	}
	subnetJson := make([]byte, 2000)
	n, err := subnetConfigFile.Read(subnetJson)
	if err != nil {
		return err
	}

	err = json.Unmarshal(subnetJson[:n], ipam.Subnets)
	if err != nil {
		log.Errorf("Error dump allocation info, %v", err)
		return err
	}
	return nil
}

func (ipam *IPAM) dump() error {
	ipamConfigFileDir, _ := path.Split(ipam.SubnetAllocatorPath)
	if _, err := os.Stat(ipamConfigFileDir); err != nil {
		if os.IsNotExist(err) {
			os.MkdirAll(ipamConfigFileDir, 0644)
		} else {
			return err
		}
	}
	subnetConfigFile, err := os.OpenFile(ipam.SubnetAllocatorPath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	defer subnetConfigFile.Close()
	if err != nil {
		return err
	}

	ipamConfigJson, err := json.Marshal(ipam.Subnets)
	if err != nil {
		return err
	}

	_, err = subnetConfigFile.Write(ipamConfigJson)
	if err != nil {
		return err
	}

	return nil
}

// Allocate 从指定网段中分配一个IP地址
func (ipam *IPAM) Allocate(subnet *net.IPNet) (ip net.IP, err error) {
	// 存放网段中地址分配信息的数组
	ipam.Subnets = &map[string]string{}

	// 从文件中加载已经分配的网段信息
	err = ipam.load()
	if err != nil {
		log.Errorf("Error dump allocation info, %v", err)
	}
	_, subnet, _ = net.ParseCIDR(subnet.String())
	// net.IPNet.Mask.size()函数会返回网段的子网掩码的总长度和网段前面的固定位的长度
	// 比如“127.0.0.0/8”网段的子网掩码是“255.0.0.0”
	// 那么 subnet.Mask.size()的返回值就是前面 255 所对应的位数和总位数，即8和24
	one, size := subnet.Mask.Size()
	// 如果之前没有分配过这个网段，则初始化网段的分配配置
	if _, exist := (*ipam.Subnets)[subnet.String()]; !exist {
		(*ipam.Subnets)[subnet.String()] = strings.Repeat("0", 1<<uint8(size-one))
	}
	// 遍历网段的位图数组
	for c := range (*ipam.Subnets)[subnet.String()] {
		// 找到数组中为“0”的项和数组序号，即可以分配的IP
		if (*ipam.Subnets)[subnet.String()][c] == '0' {
			// 设置这个为“0”的序号值为“1”，即分配这个IP
			ipalloc := []byte((*ipam.Subnets)[subnet.String()])
			ipalloc[c] = '1'
			(*ipam.Subnets)[subnet.String()] = string(ipalloc)
			ip = subnet.IP
			for t := uint(4); t > 0; t -= 1 {
				[]byte(ip)[4-t] += uint8(c >> ((t - 1) * 8))
			}
			ip[3] += 1
			break
		}
	}

	ipam.dump()
	return
}

// Release 从指定网段中释放IP
func (ipam *IPAM) Release(subnet *net.IPNet, ipaddr *net.IP) error {
	ipam.Subnets = &map[string]string{}

	_, subnet, _ = net.ParseCIDR(subnet.String())

	err := ipam.load()
	if err != nil {
		log.Errorf("Error dump allocation info, %v", err)
	}

	c := 0
	releaseIP := ipaddr.To4()
	releaseIP[3] -= 1
	for t := uint(4); t > 0; t -= 1 {
		c += int(releaseIP[t-1]-subnet.IP[t-1]) << ((4 - t) * 8)
	}

	ipalloc := []byte((*ipam.Subnets)[subnet.String()])
	ipalloc[c] = '0'
	(*ipam.Subnets)[subnet.String()] = string(ipalloc)

	ipam.dump()
	return nil
}

// ManualAllocate 方法用于手动分配指定的IP地址并保存到subnet.json
func (ipam *IPAM) ManualAllocate(subnet *net.IPNet) {
	// 基于subnet.IP和subnet计算主机位的偏移，计算出后在bitmap对应配置文件中修改对应位的值为1
	offset, err := calculateBitmapOffset(subnet.String(), subnet.IP.String())
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		return
	}
	// 存放网段中地址分配信息的数组
	ipam.Subnets = &map[string]string{}

	// 从文件中加载已经分配的网段信息
	err = ipam.load()
	if err != nil {
		log.Errorf("Error dump allocation info, %v", err)
	}
	_, subnet, _ = net.ParseCIDR(subnet.String())
	// net.IPNet.Mask.size()函数会返回网段的子网掩码的总长度和网段前面的固定位的长度
	// 比如“127.0.0.0/8”网段的子网掩码是“255.0.0.0”
	// 那么 subnet.Mask.size()的返回值就是前面 255 所对应的位数和总位数，即8和24
	one, size := subnet.Mask.Size()
	// 如果之前没有分配过这个网段，则初始化网段的分配配置
	if _, exist := (*ipam.Subnets)[subnet.String()]; !exist {
		(*ipam.Subnets)[subnet.String()] = strings.Repeat("0", 1<<uint8(size-one))
	}
	ipalloc := []byte((*ipam.Subnets)[subnet.String()])
	ipalloc[offset] = '1'
	(*ipam.Subnets)[subnet.String()] = string(ipalloc)

	ipam.dump()
	return
}

// 计算IP在子网中的bitmap偏移量
func calculateBitmapOffset(cidr string, ipStr string) (int, error) {
	// 解析CIDR
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return 0, fmt.Errorf("无效的CIDR格式: %v", err)
	}

	// 解析目标IP
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return 0, fmt.Errorf("无效的IP地址")
	}
	ip = ip.To4()
	if ip == nil {
		return 0, fmt.Errorf("仅支持IPv4地址")
	}

	// 检查IP是否属于该子网
	if !ipNet.Contains(ip) {
		return 0, fmt.Errorf("IP不在子网范围内")
	}

	// 计算网络地址和广播地址
	networkIP := ipNet.IP.To4()
	mask := ipNet.Mask

	// 将IP转换为32位整数
	ipInt := ipToUint32(ip)
	networkInt := ipToUint32(networkIP)

	// 计算主机号
	hostPart := ipInt - networkInt

	// 计算可用主机范围（排除网络地址和广播地址）
	ones, bits := mask.Size()
	totalHosts := 1 << (uint32(bits) - uint32(ones))
	if hostPart == 0 || hostPart == uint32(totalHosts-1) {
		return 0, fmt.Errorf("IP是网络地址或广播地址")
	}

	// bitmap偏移量（从0开始）
	offset := int(hostPart) - 1
	return offset, nil
}

// 将IPv4地址转换为32位整数
func ipToUint32(ip net.IP) uint32 {
	ip = ip.To4()
	return uint32(ip[0])<<24 | uint32(ip[1])<<16 | uint32(ip[2])<<8 | uint32(ip[3])
}
