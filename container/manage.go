package container

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"os"
	"strconv"
	"syscall"
)

// StopContainer 停止正在运行的容器
func StopContainer(containerName string) {
	containerInfo, err := getContainerInfoByName(containerName)
	pid := containerInfo.Pid
	if err != nil {
		log.Errorf("获取容器信息 %s 异常 %v", containerName, err)
		return
	}
	pidInt, _ := strconv.Atoi(pid) // string 转换 int
	// 中止进程
	// SIGTERM：终止信号，通常用于请求程序优雅地终止。 -15
	// SIGKILL：强制终止信号，无法被捕获或忽略，立即终止进程。 -9
	if err := syscall.Kill(pidInt, syscall.SIGTERM); err != nil {
		log.Errorf("停止容器%s的进程%d中止异常 %v", containerName, pidInt, err)
		return
	}

	if containerInfo.Status != STOP {
		// 更新配置文件中的容器信息
		containerInfo.Status = STOP
		containerInfo.Pid = "-"
		err = updateContainerInfoByName(&containerInfo)
		if err != nil {
			log.Errorf("更新容器%s信息异常 %v", containerName, err)
			return
		}

		fmt.Printf("容器: %s, ID: %s, 已进入%s\n", containerName, containerInfo.Id, containerInfo.Status)
	} else {
		fmt.Printf("容器: %s, ID: %s, 已为%s\n", containerName, containerInfo.Id, containerInfo.Status)
	}
}

// RemoveContainer 删除容器
func RemoveContainer(containerName string) {
	containerInfo, err := getContainerInfoByName(containerName)
	if err != nil {
		log.Errorf("获取容器信息 %s 异常 %v", containerName, err)
		return
	}
	if containerInfo.Status != STOP {
		log.Errorf("无法删除正在运行的容器")
		return
	}
	// 拼接配置文件路径
	dirURL := fmt.Sprintf(DefaultInfoPath, containerName)
	if err := os.RemoveAll(dirURL); err != nil {
		log.Errorf("删除配置文件 %s 异常 %v", dirURL, err)
		return
	}
	DeleteWorkSpace(containerInfo.Volume, containerName)
	fmt.Printf("容器: %s, ID: %s, 已删除\n", containerName, containerInfo.Id)
}
