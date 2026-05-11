package common

import (
	"strings"
)

// GetLocationByIP 解析 IP 地址 (Mock实现，保证编译通过)
func GetLocationByIP(ip string) string {
	if ip == "127.0.0.1" || ip == "::1" || strings.HasPrefix(ip, "192.168.") {
		return "内网IP"
	}
	return "未知位置"
}

// InitLocationService 初始化地理位置服务 (空实现)
func InitLocationService() {}
