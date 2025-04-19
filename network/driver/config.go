package driver

type DriverType string

// 驱动类型目前仅支持bridge、host
const (
	Bridge DriverType = "bridge"
	Host   DriverType = "host"
)

type Driver struct {
	DriverName string     // 驱动名称，如果有的话一般与网络名称同名
	DriverType DriverType // 驱动类型
}
