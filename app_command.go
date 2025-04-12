package main

import (
	"fmt"
	"fockker/container"
	"github.com/urfave/cli"
	"os"
	"syscall"
)

// InitCommand 不可显式调用。容器在执行/proc/self/exe后触发的方法
var InitCommand = cli.Command{
	Name: "init",
	Action: func(c *cli.Context) error {
		err := container.RunContainerInitProcess()
		if err != nil {
			nowPath, _ := os.Getwd()
			_ = syscall.Unmount(nowPath, syscall.MNT_DETACH)
		}
		return err
	},
}

// RunCommand 用户显式调用的方法。基于镜像运行容器
var RunCommand = cli.Command{
	Name:  "run",
	Usage: `基于镜像创建一个容器，包含namespace隔离、cgroup资源限制：fockker run -it [image] [command]`,
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "it",
			Usage: `开启终端`,
		},
		cli.BoolFlag{
			Name:  "d",
			Usage: `后台运行`,
		},
		cli.StringFlag{
			Name:  "name",
			Usage: `容器名称`,
		},
	},
	Action: func(c *cli.Context) error {
		if len(c.Args()) < 1 {
			return fmt.Errorf(`缺少command参数`)
		}

		// 提取输入的参数
		var cmdArry []string // cmd参数列表
		var imgName string   // 镜像名称
		for _, arg := range c.Args() {
			cmdArry = append(cmdArry, arg)
		}
		imgName = cmdArry[0]
		cmdArry = cmdArry[1:]

		// 入参解析
		createTTY := c.Bool("it")         // 是否创建可交互终端
		detach := c.Bool("d")             // 是否分离父子进程（即后台运行）
		containerName := c.String("name") // 容器运行名称

		if createTTY && detach {
			return fmt.Errorf(`不可同时指定 'it' 创建终端 与 'd' 后台运行`)
		}
		RunC(cmdArry, imgName, containerName, createTTY)
		return nil
	},
}
