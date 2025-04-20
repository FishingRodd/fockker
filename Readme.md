# 引言

参考：[自己动手写docker](https://github.com/xianlubird/mydocker)

重新整理了一下项目结构，项目中添加了大量中文注释。

# 环境
开发环境基于Ubuntu 24.10、Kernel 6.11.0-21-generic

cgroup版本为cgroup2fs


# 介绍

基于 Go 1.23.0 实现的轻量级容器引擎，自研学习用。

存储：联合文件系统采用overlayfs、宿主机与容器间挂载采用bind。

网络：支持创建Bridge类型网络，在使用本cli时会自动生成一个fockker0的bridge网络，类似于docker0实现。

- 不得创建网段重复的容器网络

守护进程：每个detach容器启动时都会附带一个Daemon进程，用于监听容器的运行情况
 
# 获取

```sh
git clone https://github.com/FishingRodd/fockker.git
```

# 项目结构

```
│  go.mod
│  go.sum
│  main.go                  全APP入口文件
│  app_command.go           CLI定义入口
│  run.go                   统一运行入口
│
├─container                 容器模块
│       config.go           统一管理容器模块下的配置信息
│       init.go             负责容器进程的创建、初始化
│       list.go             负责容器信息的获取、更新、删除
│       manage.go           负责容器运行时的停止、删除
│       volume.go           负责容器文件系统挂载的、创建、删除
│       daemon.go           负责监听detach容器的运行情况
│
├─network                   网络模块
│  │  config.go             统一管理网络模块下的配置信息
│  │  endpoint.go           负责配置网络端点IP与路由
│  │  main.go               开放的网络实现方法，支持初始化网络、创建网络、删除网络、连接网络、断开网络
│  │  network.go            负责具体的网络方法，细致的描述了上述network对象的操作
│  │
│  ├─driver                 驱动模块
│  │      config.go         统一管理驱动模块下的配置信息
│  │      driver.go         统一驱动动作，触发不同驱动类型的方法
│  │      bridge.go         bridge驱动类型的方法实现
│  │      interface.go      负责接口操作
│  │
│  ├─ipam                   IP地址管理模块(IP Address Management)
│  │      config.go         统一管理IP模块下的配置信息
│  │      ipam.go           负责IP的分配、释放、配置文件管理
│  │
│  └─iptables               IPTables模块
│          iptables.go      负责管理内外主机的通信策略
└─nsenter
        config.go           统一管理nsenter模块下的配置信息
        main.go             cgo实现的基于setns进入容器namespace
```
`fockker`运行状态信息结构
```sh
/var/run/fockker/
├── network
│  └─ fockker0
│     ├─ config.json
│     └─ subnet.json
└─ testContainer
   ├─ config.json
   └─ container.log
```
挂载系统结构
```sh
/var/root/
 ├── busybox
 │    ├── bin
 │    ├── dev
 │    ├── etc
 │    ├── home
 │    ├── lib
 │    ├── lib64 -> lib
 │    ├── proc
 │    ├── root
 │    ├── sys
 │    ├── tmp
 │    ├── usr
 │    └── var
 ├── mnt
 │    └── testContainer
 │        ├── hello.txt
 │        ├── bin
 │        ├── dev
 │        ├── etc
 │        ├── home
 │        ├── lib
 │        ├── lib64 -> lib
 │        ├── proc
 │        ├── root
 │        ├── sys
 │        ├── tmp
 │        ├── usr
 │        └── var
 ├── workLayer
 │    └── testContainer
 │        └── work
 └── writeLayer
      └── testContainer
          ├── hello.txt
          └── root
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

7. 查看当前容器网络

```sh
fockker network ls
Name        IpRange          Type
fockker0    192.168.0.1/24   bridge
```

8. 创建容器网络
```sh
fockker network create testNetwork
网络: testNetwork, 创建成功
```

9. 创建容器网络
```sh
fockker network create testNetwork
网络: testNetwork, 删除成功
```

10. 基于容器网络的容器端口映射
```sh
fockker run --it --name testContainer --p 80:80 busybox sh
容器 testContainer 启动成功
/ # 
```
容器192.168.0.2开启80端口
```sh
/ # nc -lp 80
```
宿主机访问容器80端口
```sh
[~]$ telnet 192.168.0.2 80
```
继续在宿主机的交互页面输入`hello container`并回车，此时查看容器的`nc`响应
```sh
/ # nc -lp 80
hello container
```

11. 容器资源限制，设置最大内存100m
```sh
fockker run --it --name testContainer --m 100m busybox sh
容器 testContainer 启动成功
/ # 
```