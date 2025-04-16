# 引言

参考：[自己动手写docker](https://github.com/xianlubird/mydocker)

重新整理了一下项目结构，项目中添加了大量中文注释。

# 介绍

基于 Go 1.23.0 实现的轻量级容器引擎，自研学习用。

存储：联合文件系统采用overlayfs、宿主机与容器间挂载采用bind

# 获取

```sh
git clone https://github.com/FishingRodd/fockker.git
```

# 项目结构

```
│  go.mod
│  go.sum
│  main.go          全APP入口文件
│  app_command.go   CLI定义入口
│  run.go           统一运行入口
│
└─container
        config.go   统一保存和管理模块下的配置信息
        init.go     负责容器进程的创建、初始化
        list.go     负责容器信息的获取、更新、删除
        manage.go   负责容器运行时的停止、删除
        volume.go   负责容器文件系统挂载的、创建、删除
```

# 构建

执行以下命令后，本地目录会生成一个`fockker`可执行文件。

```sh
go mod tidy
go build .
```

# 帮助

```sh
fockker help
```

# 使用

1. 创建一个提供可交互终端的、宿主机/test挂载容器/test路径的、基于路径/root/busybox下的镜像、名称为testContainer的容器

```sh
fockker run -it --name testContainer --v /test:/test busybox sh
容器 testContainer 启动成功
```

2. 创建一个后台运行`top -b`的、名称为testContainer的容器

```sh
fockker run --d --name testContainer busybox top -b
容器 testContainer 启动成功
```

3. 查看正在运行的容器信息

```sh
fockker ps
ID           NAME             PID         STATUS      COMMAND     CREATED
7120030368   testContainer    -           stopped     top-b       2025-04-16 06:35:24
```

4. 停止正在运行的容器

```sh
fockker stop testContainer
容器: testContainer, ID: 5213989969, 已进入stopped
```

5. 删除停止运行的容器

```sh
fockker rm testContainer
容器: testContainer, ID: 5213989969, 已删除
```

6. 进入正在运行的容器

```sh
fockker exec testContainer sh
```
