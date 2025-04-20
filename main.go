package main

import (
	"fmt"
	"fockker/constants"
	"fockker/network"
	_ "fockker/nsenter" // nsenter引用(必要)
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"os"
	"path/filepath"
	"runtime"
)

func main() {
	app := cli.NewApp()
	app.Name = constants.AppName
	app.Usage = constants.Usage

	app.Commands = []cli.Command{
		InitCommand,    // 容器初始化
		RunCommand,     // 容器启动
		ListCommand,    // 容器状态信息
		StopCommand,    // 容器停止
		RemoveCommand,  // 容器删除
		ExecCommand,    // 容器执行
		LogCommand,     // 容器日志
		NetwormCommand, // 容器网络
		DaemonCommand,  // Daemon进程
		// TODO BuildCommand 容器构建
	}

	// 设置日志输出
	app.Before = func(ctx *cli.Context) error {
		// 设置异常日志输出格式
		logInit()
		// 容器网络初始化
		network.InitNetwork()
		return nil
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func logInit() {
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)
	log.SetReportCaller(true) // 启用调用者信息
	log.SetFormatter(&log.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05", // 等价于 %(asctime)s
		FullTimestamp:   true,
		CallerPrettyfier: func(f *runtime.Frame) (string, string) { // 处理调用者信息
			filename := filepath.Base(f.File)     // 获取文件名
			funcName := filepath.Base(f.Function) // 获取函数名
			// 格式化为 [filename:行号:funcName]
			return "", fmt.Sprintf(" [%s:%d:%s]", filename, f.Line, funcName)
		},
		// 自定义格式
		ForceColors:  true,
		ForceQuote:   true,
		DisableQuote: false,
	})
}
