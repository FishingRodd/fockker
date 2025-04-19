package container

import (
	"fmt"
	"fockker/nsenter"
	log "github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"strconv"
	"strings"
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
	// 从网络中断开连接（容器进入STOP状态时就已会自动删除veth接口）
	// network.DisconnectFromNetwork(containerInfo.NetworkName, containerInfo.Id)
	// 拼接配置文件路径
	dirURL := fmt.Sprintf(DefaultInfoPath, containerName)
	if err := os.RemoveAll(dirURL); err != nil {
		log.Errorf("删除配置文件 %s 异常 %v", dirURL, err)
		return
	}
	DeleteWorkSpace(containerInfo.Volume, containerName)
	fmt.Printf("容器: %s, ID: %s, 已删除\n", containerName, containerInfo.Id)
}

// ExecContainer 在容器中执行命令
func ExecContainer(containerName string, cmdArry []string) {
	containerInfo, err := getContainerInfoByName(containerName)
	pid := containerInfo.Pid

	if err != nil {
		log.Errorf("获取容器信息 %s 异常 %v", containerName, err)
		return
	}

	// 拼接command参数
	cmdStr := strings.Join(cmdArry, " ")
	// 预定义command
	cmd := exec.Command("/proc/self/exe", "exec") // 传递exec，会在容器进程内再运行一次exec方法
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// 通过环境变量向cgo定义的nsenter传递参数
	_ = os.Setenv(nsenter.EnvExecPid, pid)
	_ = os.Setenv(nsenter.EnvExecCmd, cmdStr)
	// 根据PID获取进程的environments。将 当前环境变量、容器内环境变量 合并添加到command
	containerEnvs := getEnvsByPid(pid)
	cmd.Env = append(os.Environ(), containerEnvs...)
	// 启动command
	if err := cmd.Run(); err != nil {
		log.Errorf("执行容器 %s, PID: %s, 异常: %v", containerName, pid, err)
	}
}

// 根据进程PID获取environments
func getEnvsByPid(pid string) []string {
	path := fmt.Sprintf("/proc/%s/environ", pid)
	contentBytes, err := os.ReadFile(path)
	if err != nil {
		log.Errorf("读取proc文件 %s 异常 %v", path, err)
		return nil
	}
	// env的分隔符是 \u0000 ，使用split分割为[]string
	envs := strings.Split(string(contentBytes), "\u0000")
	return envs
}
