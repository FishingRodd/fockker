package container

var (
	RootPath       string = "/root"
	ImgLayerPath   string = RootPath + "/%s"            // 镜像存储路径，%s为镜像名
	WriteLayerPath string = RootPath + "/writeLayer/%s" // 容器层文件路径，%s为容器名
	WorkLayerPath  string = RootPath + "/workLayer/%s"  // 工作目录存储路径，%s为容器名
	MountPath      string = RootPath + "/mnt/%s"        // 联合挂载点路径，%s为容器名
)

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
// └── writeLayer
//     └── test_container
// │       ├── hello.txt
//         └── root
