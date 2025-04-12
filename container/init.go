package container

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

// NewContainerProcess 创建容器进程
func NewContainerProcess(imgName string, containerName string, createTTY bool) (*exec.Cmd, *os.File) {
	// 容器进程与宿主机进程通过管道互相传递参数。容器读，宿主写
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		log.Errorf("管道创建异常 %v", err)
		return nil, nil
	}

	initCmd, err := os.Readlink("/proc/self/exe")
	if err != nil {
		log.Errorf("获取初始化进程异常 %v", err)
		return nil, nil
	}

	// 创建容器进程
	cmd := exec.Command(initCmd, "init") // 通过/proc/self/exe再调用自身并传递init，启动容器应用的运行
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | // 主机名与域名隔离；隔离hostname 和 domainname
			syscall.CLONE_NEWNET | // 网络命名空间隔离；网络设备隔离
			syscall.CLONE_NEWPID | // PID进程隔离；独立PID空间
			syscall.CLONE_NEWIPC | // 消息队列隔离；隔离System V IPC 或 POSIX
			syscall.CLONE_NEWNS, // 挂载命名空间隔离；mount挂载视图独立
	}
	if createTTY {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	} else {

	}

	// 容器内通过额外的文件描述符去访问这个read管道；一般文件的描述符有3个，这里手动添加了一个
	cmd.ExtraFiles = []*os.File{readPipe} // 在Linux中，很多资源（如管道、套接字、设备等）均被视为文件
	// 在宿主机使用AUFS初始化容器内的文件系统
	err = NewWorkSpace(imgName, containerName)
	if err != nil {
		// 方法内层会抛出对应error
		return nil, nil
	}
	//cmd.Env
	//cmd.Dir工作目录
	return cmd, writePipe
}

// RunContainerInitProcess 初始化容器进程
func RunContainerInitProcess() error {
	cmdArry := readCommand()
	if cmdArry == nil || len(cmdArry) == 0 {
		return fmt.Errorf(`运行容器参数时异常, command参数为空`)
	}
	err := setupMount()
	if err != nil {
		log.Errorf("%v", err)
		return err
	}

	// 寻找命令绝对路径避免异常，例如ll实际为/usr/bin/ls -l
	path, err := exec.LookPath(cmdArry[0])
	if err != nil {
		log.Errorf("Exec loop path error %v", err)
		return err
	}

	// 通过syscall.Exec方法 运行容器需要启动的进程/应用，并将该进程PID与init初始化的PID替换
	// 例：fockker run -it ll 一开始init进程（/proc/self/init）必定为隔离空间内第一个进程，而此处的Exec将ll替换了init进程，所以使用ps查看进程时会发现ll的PID为1
	if err := syscall.Exec(path, cmdArry[0:], os.Environ()); err != nil {
		log.Errorf(err.Error())
	}
	return nil
}

// 从文件描述符获取read管道并读取args
func readCommand() []string {
	pipe := os.NewFile(uintptr(3), "pipe")
	defer func() {
		_ = pipe.Close()
	}()
	msg, err := io.ReadAll(pipe)
	if err != nil {
		log.Errorf(`初始化read管道异常 %v`, err)
		return nil
	}
	msgStr := string(msg)
	return strings.Split(msgStr, " ")
}

// 设置容器环境的初始挂载
func setupMount() error {
	nowPath, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("当前路径获取异常: %v", err)
	}

	// 使用pivotRoot实现基于根的完整隔离
	err = pivotRoot(nowPath)
	if err != nil {
		return fmt.Errorf("pivotRoot挂载失败: %v", err)
	}

	// syscall.MS_NOEXEC 在本文件系统中不允许运行其他程序。
	// - 即挂载后的文件系统中的可执行文件无法被执行。
	// syscall.MS_NOSUID 在本系统中运行程序的时候， 不允许set-user-id和set-group-id。
	// - 即使某个文件具有suid或sgid权限，也不会以文件拥有者的权限执行。
	// syscall.MS_NODEV 用于禁止在挂载的文件系统中使用设备文件。
	// - 可以避免对设备文件的访问，增强文件系统的安全性。
	defaultMountFlags := syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_NODEV

	// 下面基于已实现隔离的进程中再挂载文件系统，使只对自身namespace内容可见
	// 挂载proc
	err = syscall.Mount("proc", "/proc", "proc", uintptr(defaultMountFlags), "")
	if err != nil {
		return fmt.Errorf("proc挂载异常: %v", err)
	}
	// 挂载tmpfs
	err = syscall.Mount("tmpfs", "/dev", "tmpfs", syscall.MS_NOSUID|syscall.MS_STRICTATIME, "mode=755")
	// 将tmpfs挂载到/dev目录可为容器提供快速、临时且安全的环境，它会将文件存储在内存中，避免不必要的磁盘I/O
	if err != nil {
		return fmt.Errorf("tmpfs挂载异常: %v", err)
	}

	return nil
}

// 改变当前进程的根文件系统
func pivotRoot(path string) error {
	// 虽然已经使用了Mount Namespace实现了挂载隔离，但容器内部的/目录仍是宿主机的根文件系统的一部分
	// 使用pivoRoot实现容器根文件系统在逻辑上与宿主机完全分离，实现真正的根隔离

	// 基于已实现隔离的再挂载当前路径，实现视图隔离
	if err := syscall.Mount(path, path, "bind", syscall.MS_BIND|syscall.MS_REC, ""); err != nil { // MS_BIND 创建绑定挂载，MS_REC 递归处理子挂载点
		return fmt.Errorf("mount rootfs to itself error: %v", err)
	}

	// 创建pivot文件夹，作为旧根系统的存放位置
	pivotDir := filepath.Join(path, ".pivot_root")
	if err := os.Mkdir(pivotDir, 0777); err != nil {
		return err
	}
	// systemd 加入linux之后, mount namespace 就变成 shared by default, 所以必须显式声明要这个新的mount namespace独立。
	if err := syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, ""); err != nil {
		return fmt.Errorf("设置私有传播类型失败: %v", err)
	}
	// 将当前进程的根文件系统切换为 path，旧根移动到 pivotDir
	if err := syscall.PivotRoot(path, pivotDir); err != nil {
		return fmt.Errorf("根切换错误 %v", err)
	}
	// cd切换工作目录到新挂载的根，避免进程的当前工作目录停留在旧根文件系统中
	if err := syscall.Chdir("/"); err != nil {
		return fmt.Errorf("chdir / %v", err)
	}
	// 卸载并删除旧根文件系统
	pivotDir = filepath.Join("/", ".pivot_root")
	if err := syscall.Unmount(pivotDir, syscall.MNT_DETACH); err != nil {
		return fmt.Errorf("unmount pivot_root dir %v", err)
	}
	return os.Remove(pivotDir)

	// 在pivotroot后，nowpath（例）的/app指向，就从旧根上 app目录 指向为了新根 app目录。
	// ( 依旧在新根中使用/app路径的话，在旧根上就表现为/app/app )
	// 所以需要基于新根，切换新的工作目录，即 cd / 并且重新定义pivotDir变量，再执行卸载删除操作
}
