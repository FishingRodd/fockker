package main

import (
	"fmt"
	_ "fockker/nsenter" // nsenter引用(必要)
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"os"
	"path/filepath"
	"runtime"
)

const (
	appName = "fockker"
	usage   = `fockker是一个轻量的容器引擎实现`
)

func main() {
	app := cli.NewApp()
	app.Name = appName
	app.Usage = usage

	app.Commands = []cli.Command{
		InitCommand,
		RunCommand,
		ListCommand,
		StopCommand,
		RemoveCommand,
		ExecCommand,
	}

	// 设置日志输出
	app.Before = func(ctx *cli.Context) error {
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
		return nil
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
