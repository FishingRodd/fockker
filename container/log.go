package container

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
)

// GetLogContent 根据文件获取日志内容，输出到屏幕
func GetLogContent(containerName string) {
	dirURL := fmt.Sprintf(DefaultInfoPath, containerName)
	logFilePath := dirURL + LogFileName
	file, err := os.Open(logFilePath)
	// 从配置文件获取容器日志路径

	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Errorf("日志文件 %s 关闭异常 %v", logFilePath, err)
		}
	}(file)

	if err != nil {
		log.Errorf("日志文件 %s 打开异常 %v", logFilePath, err)
		return
	}
	content, err := io.ReadAll(file)
	if err != nil {
		log.Errorf("日志文件 %s 读取异常 %v", logFilePath, err)
		return
	}

	_, err = fmt.Fprint(os.Stdout, string(content))
	if err != nil {
		log.Errorf("日志文件 %s 标准输出异常 %v", logFilePath, err)
		return
	}
}

// CreateLogFile 根据容器名创建日志文件
func CreateLogFile(containerName string) (*os.File, error) {
	dirPath := fmt.Sprintf(DefaultInfoPath, containerName)
	if err := os.MkdirAll(dirPath, 0622); err != nil {
		log.Errorf("日志配置路径 %s 创建异常 %v", dirPath, err)
		return nil, nil
	}
	// 配置日志文件路径
	stdLogFilePath := dirPath + LogFileName
	stdLogFile, err := os.Create(stdLogFilePath)
	return stdLogFile, err
}
