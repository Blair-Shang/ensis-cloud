package httpapi

import "strings"

// 同域路径前缀（生产挂在 https://www.niuma007.com 下，避免占官网根路径）。
//
// 工程根 /niuma 是产品命名空间；其下再分 API 与 Ops：
//
//	/niuma/cloud  Identity / Feedback / Updates / Ops API + healthz
//	/niuma/ops    Admin SPA
const (
	RootPrefix  = "/niuma"
	CloudPrefix = "/niuma/cloud"
	OpsPrefix   = "/niuma/ops"
)

func isCloudAPIPath(path string) bool {
	return path == CloudPrefix+"/healthz" || strings.HasPrefix(path, CloudPrefix+"/api/")
}

func isOpsPath(path string) bool {
	return path == OpsPrefix || strings.HasPrefix(path, OpsPrefix+"/")
}

func isServiceHintPath(path string) bool {
	switch path {
	case "/", "", RootPrefix, RootPrefix + "/":
		return true
	default:
		return false
	}
}
