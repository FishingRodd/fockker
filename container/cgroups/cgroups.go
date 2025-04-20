package cgroups

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"os"
	"path"
	"strconv"
)

const (
	cgroupRoot      = "/sys/fs/cgroup" // Cgroup v2统一挂载点
	cgroupProcsFile = "cgroup.procs"   // v2进程管理文件
	cgroupMaxFile   = "memory.max"     // 内存限制文件
	cgroupCPUWeight = "cpu.weight"     // CPU权重文件
	cgroupCPUMax    = "cpu.max"        // CPU配额文件
	cgroupCPUSet    = "cpuset.cpus"    // CPU亲和性文件
)

type CgroupManager struct {
	Path     string          // cgroup相对路径
	Resource *ResourceConfig // 统一资源配置
}

type ResourceConfig struct {
	MemoryLimit string
	CPUShares   string
	CPUQuota    string
	CPUSet      string
}

func NewCgroupManager(path string) *CgroupManager {
	return &CgroupManager{
		Path: path,
	}
}

// 获取完整cgroup路径
func (c *CgroupManager) getFullPath() (string, error) {
	fullPath := path.Join(cgroupRoot, c.Path)

	// 自动创建目录
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		if err := os.MkdirAll(fullPath, 0755); err != nil {
			return "", fmt.Errorf("create cgroup dir failed: %v", err)
		}
	}
	return fullPath, nil
}

// Apply 添加进程到cgroup
func (c *CgroupManager) Apply(pid int) error {
	fullPath, err := c.getFullPath()
	if err != nil {
		return err
	}

	targetFile := path.Join(fullPath, cgroupProcsFile)
	if err := os.WriteFile(targetFile, []byte(strconv.Itoa(pid)), 0644); err != nil {
		return fmt.Errorf("failed to add process %d to cgroup: %v", pid, err)
	}
	return nil
}

// Set 设置资源限制
func (c *CgroupManager) Set(res *ResourceConfig) error {
	fullPath, err := c.getFullPath()
	if err != nil {
		return err
	}

	// 设置内存限制
	if res.MemoryLimit != "" {
		target := path.Join(fullPath, cgroupMaxFile)
		if err := os.WriteFile(target, []byte(res.MemoryLimit), 0644); err != nil {
			return fmt.Errorf("set memory limit failed: %v", err)
		}
	}

	// 设置CPU权重
	if res.CPUShares != "" {
		target := path.Join(fullPath, cgroupCPUWeight)
		if err := os.WriteFile(target, []byte(res.CPUShares), 0644); err != nil {
			return fmt.Errorf("set cpu shares failed: %v", err)
		}
	}

	// 设置CPU配额
	if res.CPUQuota != "" {
		target := path.Join(fullPath, cgroupCPUMax)
		content := fmt.Sprintf("%s 100000", res.CPUQuota) // 格式: quota period
		if err := os.WriteFile(target, []byte(content), 0644); err != nil {
			return fmt.Errorf("set cpu quota failed: %v", err)
		}
	}

	// 设置CPU亲和性
	if res.CPUSet != "" {
		target := path.Join(fullPath, cgroupCPUSet)
		if err := os.WriteFile(target, []byte(res.CPUSet), 0644); err != nil {
			return fmt.Errorf("set cpuset failed: %v", err)
		}
	}

	return nil
}

// Destroy 删除cgroup
func (c *CgroupManager) Destroy() error {
	fullPath, err := c.getFullPath()
	if err != nil {
		return err
	}

	// 释放资源限制（掠过，直接删除目录即可）
	//if err := os.WriteFile(path.Join(fullPath, cgroupMaxFile), []byte("max"), 0644); err != nil {
	//	log.Warnf("reset memory limit failed: %v", err)
	//}

	// 删除cgroup目录
	if err = os.RemoveAll(fullPath); err != nil {
		log.Errorf("remove cgroup failed: %v", err)
		return err
	}
	return nil
}
