package main

import (
	"fmt"
	"fockker/container"
	"fockker/network"
	"fockker/nsenter"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"os"
	"syscall"
)

// InitCommand 不可显式调用。容器在执行/proc/self/exe后触发的方法
var InitCommand = cli.Command{
	Name: "init",
	Action: func(context *cli.Context) error {
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
		cli.StringFlag{
			Name:  "v",
			Usage: `宿主机与容器挂载，实现持久化存储`,
		},
		cli.StringFlag{
			Name:  "net",
			Usage: `连接到容器网络`,
		},
		cli.StringFlag{
			Name:  "p",
			Usage: `宿主机与容器端口映射`,
		},
		// TODO e 指定environment
		// TODO net 加入容器网络
		// TODO start 启动进入stopped状态的容器
		// TODO cgroup资源限制
	},
	Action: func(context *cli.Context) error {
		if len(context.Args()) < 1 {
			return fmt.Errorf(`缺少command参数`)
		}

		// 提取输入的参数
		var cmdArry []string // cmd参数列表
		var imgName string   // 镜像名称
		for _, arg := range context.Args() {
			cmdArry = append(cmdArry, arg)
		}
		imgName = cmdArry[0]
		cmdArry = cmdArry[1:]

		// 入参解析
		createTTY := context.Bool("it")         // 是否创建可交互终端
		detach := context.Bool("d")             // 是否分离父子进程（即后台运行）
		containerName := context.String("name") // 容器运行名称
		volume := context.String("v")           // 宿主机与容器挂载
		portMapping := context.StringSlice("p") // 宿主机与容器端口映射
		network := context.String("net")        // 连接到容器网络

		if createTTY && detach {
			return fmt.Errorf(`不可同时指定 'it' 创建终端 与 'd' 后台运行`)
		}
		RunC(cmdArry, imgName, containerName, createTTY, volume, network, portMapping)
		return nil
	},
}

var ListCommand = cli.Command{
	Name:  "ps",
	Usage: "显示所有容器",
	Action: func(context *cli.Context) error {
		container.ListContainers()
		return nil
	},
}

var StopCommand = cli.Command{
	Name:  "stop",
	Usage: "停止正在运行的容器",
	Action: func(context *cli.Context) error {
		if len(context.Args()) < 1 {
			return fmt.Errorf("缺少容器名")
		}
		containerName := context.Args().Get(0)
		container.StopContainer(containerName)
		return nil
	},
}

var RemoveCommand = cli.Command{
	Name:  "rm",
	Usage: "删除不使用的容器",
	Action: func(context *cli.Context) error {
		if len(context.Args()) < 1 {
			return fmt.Errorf("缺少容器名")
		}
		containerName := context.Args().Get(0)
		container.RemoveContainer(containerName)
		return nil
	},
}

var ExecCommand = cli.Command{
	Name:  "exec",
	Usage: "将命令执行到容器中",
	Action: func(context *cli.Context) error {
		// 容器进程callback
		if os.Getenv(nsenter.EnvExecPid) != "" {
			log.Infof("pid callback pid %d", os.Getgid())
			return nil
		}

		if len(context.Args()) < 2 {
			return fmt.Errorf("缺少容器名或入参")
		}
		containerName := context.Args().Get(0)

		// 提取入参
		var cmdArry []string
		for _, arg := range context.Args().Tail() {
			cmdArry = append(cmdArry, arg)
		}
		container.ExecContainer(containerName, cmdArry)
		return nil
	},
}

var LogCommand = cli.Command{
	Name:  "logs",
	Usage: "打印容器日志",
	Action: func(context *cli.Context) error {
		if len(context.Args()) < 1 {
			return fmt.Errorf("请输入容器名")
		}
		containerName := context.Args().Get(0)
		container.GetLogContent(containerName)
		return nil
	},
}

var NetwormCommand = cli.Command{
	Name:  "network",
	Usage: "容器网络命令行",
	Subcommands: []cli.Command{
		{
			Name:  "create",
			Usage: "创建容器网络",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "type",
					Usage: "network driver",
				},
				cli.StringFlag{
					Name:  "subnet",
					Usage: "subnet cidr",
				},
			},
			Action: func(context *cli.Context) error {
				if len(context.Args()) < 1 {
					return fmt.Errorf("缺少网络名称")
				}
				networkName := context.Args()[0]
				netType := context.String("type")
				ipRange := context.String("subnet")
				// 未指定网段则使用默认
				var networkType network.NetworkType
				if netType == "" {
					networkType = network.Bridge
				} else {
					networkType = network.NetworkType(netType)
				}
				err := network.CreateNetwork(networkName, networkType, ipRange)
				if err != nil {
					return fmt.Errorf("网络创建异常: %v", err)
				}
				return nil
			},
		},
		{
			Name:  "ls",
			Usage: "显示当前所有容器网络",
			Action: func(context *cli.Context) error {
				network.ListNetwork()
				return nil
			},
		},
		{
			Name:  "rm",
			Usage: "删除容器网络",
			Action: func(context *cli.Context) error {
				if len(context.Args()) < 1 {
					return fmt.Errorf("缺少网络名称")
				}
				networkName := context.Args()[0]
				err := network.DistoryNetwork(networkName)
				if err != nil {
					return fmt.Errorf("网络删除异常: %v", err)
				}
				return nil
			},
		},
	},
}
