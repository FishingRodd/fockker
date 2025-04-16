# 引言

参考：[自己动手写docker](https://github.com/xianlubird/mydocker)

重新整理了一下项目结构，项目钟添加了大量中文注释。

# 介绍

基于 Go 1.23.0 实现的轻量级容器引擎，自研学习用。

存储：联合文件系统采用overlayfs、宿主机与容器间挂载采用bind

# 获取

```sh
git clone https://github.com/FishingRodd/fockker.git
```

# 构建

```sh
go build .
```

执行上述命令后，本地目录会生成一个`fockker`可执行文件。

# 帮助

```sh
fockker help
```

# 使用

创建一个提供可交互终端的、宿主机/test挂载容器/test路径的、基于路径/root/busybox下的镜像、名称为testContainer的容器
```sh
fockker run -it --name testContainer --v /test:/test busybox sh
```

