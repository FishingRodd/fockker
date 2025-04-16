package main

import (
	"fmt"
	"fockker/container"
	log "github.com/sirupsen/logrus"
	"os"
	"strings"
)

// RunC 根据入参运行容器进程
func RunC(cmdArry []string, imgName string, containerName string, createTTY bool, volume string) {
	// 创建容器初始化进程
	processCmd, writePipe := container.NewContainerProcess(imgName, containerName, createTTY, volume)
	if processCmd == nil {
		log.Errorf(`容器初始化进程异常`)
		return
	}

	if err := processCmd.Start(); err != nil {
		log.Errorf(`容器初始化进程启动失败: %v`, err)
		return
	}

	// 保存容器信息
	containerName, _, err := container.RecordContainerInfo(processCmd.Process.Pid, cmdArry, containerName, volume)
	if err != nil {
		log.Errorf("保存容器信息异常 %v", err)
		return
	}

	// cgroup限制

	// 容器进程初始化启动完成后，通过管道向其发送args参数（如top、ls -l等用户在run输入的参数）
	sendInitCommand(cmdArry, writePipe)

	if createTTY {
		// 创建了可交互式终端时，宿主机进程与容器进程存在父子关系，父宿主机需要等待子容器退出终端，即 cmd.Wait()
		_ = processCmd.Wait()
		container.DeleteWorkSpace(volume, containerName)
		log.Infof(`容器 %s 退出成功`, containerName)
	}
	fmt.Printf("容器 %s 启动成功\n", containerName)
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
