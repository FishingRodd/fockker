package iptables

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"net"
	"os/exec"
	"strings"
)

func InnerToOuter(bridgeName string, subnet *net.IPNet) error {
	// MASQUERADE 允许容器主机通过宿主机IP访问外部网络
	iptablesCmd := fmt.Sprintf("-t nat -A POSTROUTING -s %s ! -o %s -j MASQUERADE", subnet.String(), bridgeName)
	cmd := exec.Command("iptables", strings.Split(iptablesCmd, " ")...)
	//err := cmd.Run()
	output, err := cmd.Output()
	if err != nil {
		log.Errorf("iptables设置异常: %v", output)
	}
	return err
}

func OuterToInner(leftPort string, rightPort string, ipAddress string) error {
	// DNAT 允许外部请求访问内网的特定服务
	iptablesCmd := fmt.Sprintf("-t nat -A PREROUTING -p tcp -m tcp --dport %s -j DNAT --to-destination %s:%s", leftPort, ipAddress, rightPort)
	cmd := exec.Command("iptables", strings.Split(iptablesCmd, " ")...)
	//err := cmd.Run()
	output, err := cmd.Output()
	if err != nil {
		log.Errorf("iptables设置异常: %v", output)
	}
	return err
}
