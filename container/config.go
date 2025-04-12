package container

var (
	RootPath       string = "/root"
	MountPath      string = "/root/mnt/%s"        // 联合挂载点路径，%s为容器名
	WriteLayerPath string = "/root/writeLayer/%s" // 容器层文件路径，%s为容器名
)
