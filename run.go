package main

import (
	"fmt"
	"fockker/constants"
	"fockker/container"
	"fockker/container/cgroups"
	"fockker/network"
	log "github.com/sirupsen/logrus"
	"os"
	"strconv"
	"strings"
)

// RunC 根据入参运行容器进程
func RunC(cmdArry []string, imgName string, containerName string, createTTY bool, volume string, networkName string,
	portMapping []string, envSlice []string, resourceConf *cgroups.ResourceConfig) {
	// 判断containerName是否重复
	_, err := container.GetContainerInfoByName(containerName)
	if err == nil {
		fmt.Printf("容器 %s 创建失败: 该名称已存在\n", containerName)
		return
	}
	// 创建容器初始化进程
	processCmd, writePipe := container.NewContainerProcess(imgName, containerName, createTTY, volume, envSlice)
	if processCmd == nil {
		log.Errorf(`容器初始化进程异常`)
		return
	}

	if err := processCmd.Start(); err != nil {
		log.Errorf(`容器初始化进程启动失败: %v`, err)
		return
	}

	if networkName == "" {
		// 加入默认网络
		networkName = network.DefaultBridgeName
	}

	// 保存容器信息
	containerInfo, err := container.RecordContainerInfo(processCmd.Process.Pid, cmdArry, containerName, volume, networkName, portMapping)
	if err != nil {
		log.Errorf("保存容器信息异常 %v", err)
		return
	}
	// 加入网络
	network.ConnectToNetwork(networkName, containerInfo.Id, containerInfo.PortMapping, containerInfo.Pid)

	// cgroup限制
	cgroupPath := fmt.Sprintf("%s/%s", constants.AppName, containerName)
	cgroupManager := cgroups.NewCgroupManager(cgroupPath)
	if createTTY {
		// 未detach分离的可以通过父进程defer管理cgroup
		defer func(cgroupManager *cgroups.CgroupManager) {
			err = cgroupManager.Destroy()
			if err != nil {
				log.Errorf("cgroup释放异常")
			}
		}(cgroupManager)
	} else {
		// 已经实现detach分离的容器进程由pid 1的init进程管理，这里采用信号管理该进程
		// 启动一个daemon进程，监听容器的系统信号，回收cgroupPath
		go container.StartDaemon(processCmd.Process.Pid, cgroupPath, containerName)
	}
	_ = cgroupManager.Set(resourceConf)
	pid, _ := strconv.Atoi(containerInfo.Pid)
	_ = cgroupManager.Apply(pid)

	// 容器进程初始化启动完成后，通过管道向其发送args参数（如top、ls -l等用户在run输入的参数）
	sendInitCommand(cmdArry, writePipe)

	if createTTY {
		// 创建了可交互式终端时，宿主机进程与容器进程存在父子关系，父宿主机需要等待子容器退出终端，即 cmd.Wait()
		_ = processCmd.Wait()
		containerInfo.Status = container.STOP
		_ = container.UpdateContainerInfoByName(containerInfo)
		container.RemoveContainer(containerName)
		//fmt.Printf("容器 %s 退出成功\n", containerName)
	} else {
		fmt.Printf("容器 %s 启动成功\n", containerName)
	}
}

func sendInitCommand(cmdArry []string, writePipe *os.File) {
	command := strings.Join(cmdArry, " ")
	_, err := writePipe.WriteString(command)
	if err != nil {
		log.Errorf(`write管道写入异常 -- %s`, command)
		return
	}
	err = writePipe.Close()
	if err != nil {
		log.Errorf(`write管道关闭异常`)
		return
	}
}
