package container

import (
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"
)

// ListContainers 获取配置路径下的全部容器信息
func ListContainers() {
	dirPath := fmt.Sprintf(DefaultInfoPath, "")
	dirPath = dirPath[:len(dirPath)-1] // 去掉最后的/斜杠
	files, err := os.ReadDir(dirPath)  // 读取该路径下所有文件
	if err != nil {
		log.Errorf("读取目录 %s 异常 %v", dirPath, err)
		return
	}

	// 保存容器信息
	var containers []*ContainerInfo
	for _, file := range files {
		if file.Name() == "network" {
			continue
		}
		// 根据fileInfo读取文件，获取所有container信息
		tmpContainer, err := getContainerInfo(file)
		if err != nil {
			log.Errorf("获取容器信息异常 %v", err)
			continue
		}
		// 检查RUNNING状态的容器是否仍然存活
		if tmpContainer.Status == RUNNING {
			// 不存活则删除
			pid, _ := strconv.Atoi(tmpContainer.Pid)
			err = syscall.Kill(pid, 0)
			if errors.Is(err, syscall.ESRCH) {
				// 返回的错误是os.ErrProcess, 表示进程不存在
				tmpContainer.Status = STOP
				err = UpdateContainerInfoByName(tmpContainer)
				if err != nil {
					log.Errorf("更新容器%s信息异常 %v", tmpContainer.Name, err)
					continue
				}
				RemoveContainer(tmpContainer.Name)
				continue
			}
		}
		// 添加到containers
		containers = append(containers, tmpContainer)
	}

	// 格式化输出
	w := tabwriter.NewWriter(os.Stdout, 12, 1, 3, ' ', 0)
	_, err = fmt.Fprint(w, "ID\tNAME\tPID\tSTATUS\tCOMMAND\tCREATED\n")
	for _, item := range containers {
		_, err = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			item.Id,
			item.Name,
			item.Pid,
			item.Status,
			item.Command,
			item.CreatedTime)
	}
	if err := w.Flush(); err != nil {
		log.Errorf("容器信息刷写异常 %v", err)
	}
}

// getContainerInfo 获取容器信息
func getContainerInfo(entry os.DirEntry) (*ContainerInfo, error) {
	containerName := entry.Name()
	configFileDir := fmt.Sprintf(DefaultInfoPath, containerName)
	configFileDir = configFileDir + ConfigName

	content, err := os.ReadFile(configFileDir)
	if err != nil {
		log.Errorf("读取文件 %s 异常 %v", configFileDir, err)
		return nil, err
	}
	var containerInfo ContainerInfo
	if err := json.Unmarshal(content, &containerInfo); err != nil {
		log.Errorf("JSON反序列化异常 %v", err)
		return nil, err
	}

	return &containerInfo, nil
}

// RecordContainerInfo 记录容器信息，启用于容器创建时
func RecordContainerInfo(containerPID int, cmdArry []string, containerName string, volume string, networkName string, portMapping []string) (*ContainerInfo, error) {
	// 不指定容器名则使用ID作为容器名
	containerID := generateContainerID(10)
	if containerName == "" {
		containerName = containerID
	}
	// 初始化容器状态信息
	createTime := time.Now().Format("2006-01-02 15:04:05")
	command := strings.Join(cmdArry, " ")
	containerInfo := &ContainerInfo{
		Id:          containerID,
		Pid:         strconv.Itoa(containerPID),
		Command:     command,
		CreatedTime: createTime,
		Status:      RUNNING,
		Name:        containerName,
		Volume:      volume,
		NetworkName: networkName,
		PortMapping: portMapping,
	}
	// 序列化容器状态信息
	jsonBytes, err := json.Marshal(containerInfo)
	if err != nil {
		log.Errorf("序列化容器状态信息异常 %v", err)
		return &ContainerInfo{}, err
	}
	containerJsonInfo := string(jsonBytes) // 保存为JSON格式字符

	// 在指定路径下根据容器名创建文件夹
	dirPath := fmt.Sprintf(DefaultInfoPath, containerName)
	if err := os.MkdirAll(dirPath, 0622); err != nil {
		log.Errorf("配置路径 %s 创建异常 %v", dirPath, err)
		return &ContainerInfo{}, err
	}
	configPath := dirPath + "/" + ConfigName
	configFile, err := os.Create(configPath)

	defer func(file *os.File) {
		err := file.Close()
		if err != nil {

		}
	}(configFile)

	// error handle
	if err != nil {
		log.Errorf("创建容器状态文件 %s 异常 %v", configPath, err)
		return &ContainerInfo{}, err
	}
	// 写入配置文件
	if _, err := configFile.WriteString(containerJsonInfo); err != nil {
		log.Errorf("写入容器状态信息异常 %v", err)
		return &ContainerInfo{}, err
	}

	// 返回容器名、容器ID、err
	return containerInfo, nil
}

// 生成容器ID
func generateContainerID(n int) string {
	letterBytes := "1234567890"
	rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

// GetContainerInfoByName 根据容器名 获取 配置文件中的信息
func GetContainerInfoByName(containerName string) (ContainerInfo, error) {
	dirURL := fmt.Sprintf(DefaultInfoPath, containerName)
	configFilePath := dirURL + ConfigName
	contentBytes, err := os.ReadFile(configFilePath)
	if err != nil {
		return ContainerInfo{}, err
	}
	// 从本地配置文件中读取文件信息
	var containerInfo ContainerInfo
	if err := json.Unmarshal(contentBytes, &containerInfo); err != nil {
		return ContainerInfo{}, err
	}
	return containerInfo, nil
}

// 根据容器名和更新信息 更新 配置文件
func UpdateContainerInfoByName(newContainerInfo *ContainerInfo) error {
	// 拼接完整的配置文件路径
	configFilePath := fmt.Sprintf(DefaultInfoPath, newContainerInfo.Name) + ConfigName

	// 将更新后的结构体转换成 JSON 格式
	updatedContentBytes, err := json.Marshal(newContainerInfo)

	if err != nil {
		return err // 返回错误
	}
	// 将更新后的内容写回到配置文件
	if err = os.WriteFile(configFilePath, updatedContentBytes, 0644); err != nil {
		return err // 返回错误
	}

	return nil
}
