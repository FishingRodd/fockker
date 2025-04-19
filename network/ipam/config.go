package ipam

type IPAM struct {
	SubnetAllocatorPath string
	Subnets             *map[string]string
}
