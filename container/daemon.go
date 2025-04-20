package container

import (
	"errors"
	"fmt"
	"fockker/container/cgroups"
	log "github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

// StartDaemon 启动独立监控进程
func StartDaemon(containerPID int, cgroupPath string, containerName string) {
	// 获取当前可执行文件路径
	exePath, err := os.Executable()
	if err != nil {
		log.Errorf("get executable path failed: %v", err)
		return
	}

	// 构建监控进程命令
	cmd := exec.Command(exePath, "daemon", strconv.Itoa(containerPID), cgroupPath, containerName)

	// 分离进程属性
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true, // 创建新会话
		Pgid:   0,    // 独立进程组
		//Cloneflags: syscall.CLONE_NEWPID, // 可选：使用独立PID命名空间
	}

	// 后台运行
	if err := cmd.Start(); err != nil {
		log.Errorf("start daemon failed: %v", err)
		return
	}

	// 父进程立即退出，监控进程成为孤儿进程由init接管
	return
}

// RunDaemon 给每一个容器启动一个守护进程
func RunDaemon(pid int, cgroupPath string, containerName string) {
	_ = os.Mkdir("/daemon"+strconv.Itoa(pid), 0777)
	// 创建信号通道
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh,
		syscall.SIGTERM, // 容器终止信号
		syscall.SIGINT,  // 中断信号
		syscall.SIGCHLD, // 子进程状态变化
	)

	// 开启一个 goroutine 监控目标进程
	go func() {
		for {
			// 通过 syscall.Kill 检查目标进程是否存活
			err := syscall.Kill(pid, 0)
			if err != nil {
				// 如果返回错误，说明目标进程已终止
				if errors.Is(err, syscall.ESRCH) {
					// 执行清理操作
					cgroupManager := cgroups.NewCgroupManager(cgroupPath)
					err = cgroupManager.Destroy()
					if err != nil {
						// TODO daemon进程的日志输出定义
					}
					containerInfo, err := GetContainerInfoByName(containerName)
					if err != nil {
						log.Errorf("获取容器信息 %s 异常 %v", containerName, err)
						return
					}
					containerInfo.Status = Exit // 容器进程异常退出
					_ = UpdateContainerInfoByName(&containerInfo)
					//RemoveContainer(containerName)
					// 退出守护进程
					os.Exit(0)
				}
			}

			// 休眠一段时间，避免频繁检查
			time.Sleep(2 * time.Second)
		}
	}()

	// 主逻辑等待信号以优雅退出
	select {
	case <-sigCh:
		fmt.Println("收到退出信号，守护进程即将退出...")
		os.Exit(0)
	}
}
