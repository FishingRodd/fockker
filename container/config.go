package container

// 容器运行与挂载路径
var (
	RootPath       string = "/root"
	ImgLayerPath   string = RootPath + "/%s"            // 镜像存储路径，%s为镜像名
	WriteLayerPath string = RootPath + "/writeLayer/%s" // 容器层文件路径，%s为容器名
	WorkLayerPath  string = RootPath + "/workLayer/%s"  // 工作目录存储路径，%s为容器名
	MountPath      string = RootPath + "/mnt/%s"        // 联合挂载点路径，%s为容器名
)

// 容器运行状态与管理路径
var (
	RUNNING         string = "running"
	STOP            string = "stopped"
	Exit            string = "exited"
	DefaultInfoPath string = "/var/run/fockker/%s/"
	ConfigName      string = "config.json"
	LogFileName     string = "container.log"
)

// ContainerInfo 容器状态信息
type ContainerInfo struct {
	Pid         string   `json:"pid"`         // 容器的init进程在宿主机上的 PID
	Id          string   `json:"id"`          // 容器Id
	Name        string   `json:"name"`        // 容器名
	Command     string   `json:"command"`     // 容器内init运行命令
	CreatedTime string   `json:"createTime"`  // 创建时间
	Status      string   `json:"status"`      // 容器的状态
	Volume      string   `json:"volume"`      // 容器的数据卷
	PortMapping []string `json:"portmapping"` // 端口映射
}

// 启动容器后，挂载路径如下所示
// .
// ├── busybox
// │    ├── bin
// │    ├── dev
// │    ├── etc
// │    ├── home
// │    ├── lib
// │    ├── lib64 -> lib
// │    ├── proc
// │    ├── root
// │    ├── sys
// │    ├── tmp
// │    ├── usr
// │    └── var
// ├── mnt
// │    └── test_container
// │        ├── hello.txt
// │        ├── bin
// │        ├── dev
// │        ├── etc
// │        ├── home
// │        ├── lib
// │        ├── lib64 -> lib
// │        ├── proc
// │        ├── root
// │        ├── sys
// │        ├── tmp
// │        ├── usr
// │        └── var
// ├── workLayer
// │    └── test_container
// │        └── work
// └── writeLayer
//      └── test_container
//          ├── hello.txt
//          └── root
