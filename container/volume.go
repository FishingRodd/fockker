package container

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"os"
	"os/exec"
)

// NewWorkSpace 初始化分层文件系统
func NewWorkSpace(imgName string, containerName string) error {
	var err error
	// 镜像层
	err = CreateReadOnlyLayer(imgName)
	if err != nil {
		log.Errorf(`镜像层创建失败: %v`, err)
		return err
	}
	// 容器层
	err = CreateWriteLayer(containerName)
	if err != nil {
		log.Errorf(`容器层创建失败: %v`, err)
		return err
	}
	// 各分层系统创建完成后，通过AUFS挂载联合文件系统
	err = CreateMountPoint(imgName, containerName)
	if err != nil {
		log.Errorf(`容器层创建失败: %v`, err)
		return err
	}

	return nil
}

// CreateReadOnlyLayer 镜像层，onlyRead
func CreateReadOnlyLayer(imgName string) error {
	// 此处需要先将镜像的tar包放置在rootPath下，代码会解压并创建对应文件系统
	unTarFolderPath := RootPath + "/" + imgName + "/" // 镜像解压后的路径
	imgPath := RootPath + "/" + imgName + ".tar"      //镜像所在路径
	exists, err := PathExists(unTarFolderPath)

	if err != nil {
		log.Infof("镜像目录 %s 判断异常: %v", unTarFolderPath, err)
		return err
	}
	// 路径下不存在对应镜像目录
	if !exists {
		// 创建一个目录
		if err := os.MkdirAll(unTarFolderPath, 0622); err != nil {
			log.Errorf("镜像目录 %s 创建异常 %v", unTarFolderPath, err)
			return err
		}
		// 并且从本地路径解压
		if _, err := exec.Command("tar", "-xvf", imgPath, "-C", unTarFolderPath).CombinedOutput(); err != nil {
			log.Errorf("镜像目录 %s 解压异常 %v", unTarFolderPath, err)
			return err
		}
		// TODO 本地镜像tar包不存在走网络获取镜像tar到rootPath
	}
	return nil
}

// CreateWriteLayer 容器层，Read & Write
func CreateWriteLayer(containerName string) error {
	writePath := fmt.Sprintf(WriteLayerPath, containerName)
	if err := os.MkdirAll(writePath, 0777); err != nil {
		log.Errorf("容器目录 %s 创建异常 %v", writePath, err)
		return err
	}
	return nil
}

// CreateMountPoint 创建联合文件系统
func CreateMountPoint(imgName string, containerName string) error {
	nowMountPath := fmt.Sprintf(MountPath, containerName)
	if err := os.MkdirAll(nowMountPath, 0777); err != nil {
		log.Errorf("联合挂载目录 %s 创建异常 %v", nowMountPath, err)
		return err
	}
	writePath := fmt.Sprintf(WriteLayerPath, containerName)
	imgPath := RootPath + "/" + imgName

	// 拼接aufs挂载命令，第一个目录为rw，其余为r只读层
	dirs := "dirs=" + writePath + ":" + imgPath
	_, err := exec.Command("mount", "-t", "aufs", "-o", dirs, "none", nowMountPath).CombinedOutput()
	// 这里用syscall或unix的Mount也可以，暂没有尝试
	if err != nil {
		log.Errorf("联合文件挂载点创建异常 %v", err)
		return err
	}
	return nil
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
