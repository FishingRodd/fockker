package container

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

//// OuterFunc 外部调用的方法
//type OuterFunc interface {
//	NewWorkSpace()    // 创建容器工作目录
//	DeleteWorkSpace() // 删除容器工作目录
//}
//
//// InnerFunc 内部调用的主要方法
//type InnerFunc interface {
//	CreateReadOnlyLayer() (string, error) // 创建镜像层
//	CreateWriteLayer() (string, error)    // 创建容器层
//	CreateWorkLayer() (string, error)     // 创建工作目录
//	CreateMountPoint() (string, error)    // 创建联合挂载点
//
//	DeleteWriteLayer()           // 删除容器层
//	DeleteWorkLayer()            // 删除工作目录
//	DeleteMountPoint()           // 删除联合挂载点
//	DeleteMountPointWithVolume() // 删除联合挂载点，排除用户挂载
//}
//
//// UtilsFunc 插件方法
//type UtilsFunc interface {
//	PathExists(path string) (bool, error) // 判断路径文件是否存在
//}

// NewWorkSpace 初始化分层文件系统
func NewWorkSpace(imgName string, containerName string, volume string) error {
	// 镜像层
	imgPath, err := CreateReadOnlyLayer(imgName)
	if err != nil {
		log.Errorf(`镜像层创建失败: %v`, err)
		return err
	}
	// 容器层
	writePath, err := CreateWriteLayer(containerName)
	if err != nil {
		log.Errorf(`容器层创建失败: %v`, err)
		return err
	}
	// 工作目录层
	workPath, err := CreateWorkLayer(containerName)
	if err != nil {
		log.Errorf(`工作目录层创建失败: %v`, err)
		return err
	}

	// 各分层系统创建完成后，通过overlayfs挂载联合文件系统
	nowMountPath := fmt.Sprintf(MountPath, containerName)
	if err := os.MkdirAll(nowMountPath, 0777); err != nil {
		log.Errorf("联合挂载目录 %s 创建异常 %v", nowMountPath, err)
		return err
	}
	err = CreateMountPoint(imgPath, writePath, workPath, nowMountPath)
	if err != nil {
		log.Errorf(`挂载失败: %v`, err)
		return err
	}
	if volume != "" {
		volumePaths := strings.Split(volume, ":")
		length := len(volumePaths)
		if length == 2 && volumePaths[0] != "" && volumePaths[1] != "" {
			MountVolume(volumePaths, containerName)
			log.Infof("容器持久化路径 %q 挂载成功", volumePaths)
		} else {
			log.Infof("容器持久化挂载参数 %q 错误", volumePaths)
		}
	}

	return nil
}

// MountVolume 实现宿主机与容器内部目录挂载
func MountVolume(volumePaths []string, containerName string) {
	// 宿主机内的挂载路径
	parentPath := volumePaths[0]
	if exists, _ := PathExists(parentPath); !exists { // 路径不存在则创建
		if err := os.MkdirAll(parentPath, 0777); err != nil {
			log.Errorf("宿主机目录 %s 创建异常 %v", parentPath, err)
			return
		}
	}
	// 容器内的挂载路径
	containerUrl := volumePaths[1]
	nowMountPath := fmt.Sprintf(MountPath, containerName)
	containerVolumePath := nowMountPath + containerUrl
	if exists, _ := PathExists(containerVolumePath); !exists { // 路径不存在则创建
		if err := os.MkdirAll(containerVolumePath, 0777); err != nil {
			log.Errorf("容器目录 %s 创建异常 %v", containerVolumePath, err)
			return
		}
	}
	if err := syscall.Mount(parentPath, containerVolumePath, "bind", syscall.MS_BIND|syscall.MS_REC, ""); err != nil { // MS_BIND 创建绑定挂载，MS_REC 递归处理子挂载点
		log.Errorf("用户文件挂载点创建异常 %v", err)
	}
}

// DeleteWorkSpace 卸载并删除容器文件系统
func DeleteWorkSpace(volume, containerName string) {
	nowMountPath := fmt.Sprintf(MountPath, containerName)

	// TODO 双overlayfs BUG，发现在sendInitCommand后，mount的overlayfs就多了一个，导致一个container有两个完全一样的挂载点，在DeleteWorkSpace时删除文件目录时会显示device or resource busy
	// 可能是子进程 即容器进程也启动了一个overlayfs，可尝试把workdir移出挂载点
	_ = syscall.Unmount(fmt.Sprintf(MountPath, containerName), syscall.MNT_DETACH)

	if volume != "" {
		volumePath := strings.Split(volume, ":")
		length := len(volumePath)

		if length == 2 && volumePath[0] != "" && volumePath[1] != "" {
			DeleteMountPointWithVolume(volumePath, nowMountPath)
		} else {
			DeleteMountPoint(nowMountPath)
		}
	} else {
		DeleteMountPoint(nowMountPath)
	}
	DeleteWriteLayer(containerName)
	DeleteWorkLayer(containerName)
	// 镜像层由于是只读，此处保留并不删除
}

// CreateReadOnlyLayer 镜像层，onlyRead
func CreateReadOnlyLayer(imgName string) (string, error) {
	// 此处需要先将镜像的tar包放置在rootPath下，代码会解压并创建对应文件系统
	imgPath := fmt.Sprintf(ImgLayerPath, imgName) // 镜像解压后的路径
	tarFilePath := imgPath + ".tar"               // 镜像tar所在路径

	exists, err := PathExists(imgPath)
	if err != nil {
		log.Infof("镜像目录 %s 判断异常: %v", imgPath, err)
		return "", err
	}

	// 路径下不存在对应镜像目录
	if !exists {
		// 创建一个目录
		if err := os.MkdirAll(imgPath, 0622); err != nil {
			log.Errorf("镜像目录 %s 创建异常 %v", imgPath, err)
			return "", err
		}
		// 并且从本地路径解压
		if _, err := exec.Command("tar", "-xvf", tarFilePath, "-C", imgPath).CombinedOutput(); err != nil {
			log.Errorf("镜像目录 %s 解压异常 %v", imgPath, err)
			return "", err
		}
		// TODO 本地镜像tar包不存在，则走网络获取镜像tar到rootPath
	}
	return imgPath, nil
}

// CreateWriteLayer 容器层，Read & Write
func CreateWriteLayer(containerName string) (string, error) {
	writePath := fmt.Sprintf(WriteLayerPath, containerName)

	exists, err := PathExists(writePath)
	if err != nil {
		log.Infof("容器目录 %s 判断异常: %v", writePath, err)
		return "", err
	}

	// 路径下不存在对应容器目录
	if !exists {
		if err := os.MkdirAll(writePath, 0777); err != nil {
			log.Errorf("容器目录 %s 创建异常 %v", writePath, err)
			return "", err
		}
	}
	return writePath, nil
}

// CreateWorkLayer 工作目录层，临时
func CreateWorkLayer(containerName string) (string, error) {
	workPath := fmt.Sprintf(WorkLayerPath, containerName)

	exists, err := PathExists(workPath)
	if err != nil {
		log.Infof("系统工作目录 %s 判断异常: %v", workPath, err)
		return "", err
	}

	// 路径下不存在对应工作目录
	if !exists {
		if err := os.MkdirAll(workPath, 0777); err != nil {
			log.Errorf("系统工作目录 %s 创建异常 %v", workPath, err)
			return "", err
		}
	}
	return workPath, nil
}

// CreateMountPoint 创建联合文件系统
func CreateMountPoint(imgPath string, writePath string, workPath string, nowMountPath string) error {
	// 基于overlayfs实现联合挂载
	// lowerdir：底层目录，ro
	// upperdir：上层目录，w
	// workdir：在此处理需要的文件变更，将结果合并到最终的挂载点
	dirs := "lowerdir=" + imgPath + ",upperdir=" + writePath + ",workdir=" + workPath
	_, err := exec.Command("mount", "-t", "overlay", "-o", dirs, "overlay", nowMountPath).CombinedOutput()
	// 这里用syscall或unix的Mount也可以
	//err := syscall.Mount("overlay", nowMountPath, "overlay", 0, dirs)
	if err != nil {
		log.Errorf("系统联合文件挂载点创建异常 %v", err)
		return err
	} else {
		//log.Infof(`系统联合文件挂载点创建成功 %s`, nowMountPath)
	}
	return nil
}

// DeleteWriteLayer 删除容器层
func DeleteWriteLayer(containerName string) {
	writePath := fmt.Sprintf(WriteLayerPath, containerName)
	if err := os.RemoveAll(writePath); err != nil {
		log.Infof("删除容器层目录 %s 时异常 %v", writePath, err)
	}
}

// DeleteWorkLayer 删除工作目录
func DeleteWorkLayer(containerName string) {
	workPath := fmt.Sprintf(WorkLayerPath, containerName)

	if err := os.RemoveAll(workPath); err != nil {
		log.Infof("删除工作目录 %s 时异常 %v", workPath, err)
	}
}

// DeleteMountPoint 删除联合挂载点
func DeleteMountPoint(nowMountPath string) {
	// 使用延迟卸载，避免因挂载点繁忙导致失败
	if err := syscall.Unmount(nowMountPath, syscall.MNT_DETACH); err != nil {
		log.Errorf("卸载联合挂载点 %s 时异常: %v", nowMountPath, err)
	}
	if err := os.RemoveAll(nowMountPath); err != nil {
		log.Errorf("删除挂载点目录 %s 时异常 %v", nowMountPath, err)
	}
}

// DeleteMountPointWithVolume 删除联合挂载点，排除用户挂载
func DeleteMountPointWithVolume(volumePaths []string, nowMountPath string) {
	if err := syscall.Unmount(nowMountPath, syscall.MNT_DETACH); err != nil {
		log.Errorf("卸载联合挂载点 %s 时异常: %v", nowMountPath, err)
		return
	}

	if err := os.RemoveAll(nowMountPath); err != nil {
		log.Errorf("删除系统挂载点目录 %s 时异常 %v", nowMountPath, err)
		return
	}
}

// PathExists 判断路径下文件是否存在
func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
