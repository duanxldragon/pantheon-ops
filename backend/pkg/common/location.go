package common

import (
	"strings"
)

// GetLocationByIP 解析 IP 地址 (Mock实现，保证编译通过)。
//
// 返回稳定的 i18n key（而非本地化文案），由前端按查看者语言翻译；
// 历史数据中存量的中文文案由前端 defaultValue 回退直接展示。
func GetLocationByIP(ip string) string {
	if ip == "127.0.0.1" || ip == "::1" || strings.HasPrefix(ip, "192.168.") {
		return "location.intranet"
	}
	return "location.unknown"
}

// LocationDisplayText 把 GetLocationByIP 返回的 i18n key 还原为可读文案，
// 用于 CSV 导出等无法访问查看者语言的场景（默认回退到中文）。
// 非本组识别的值（历史存量文案）原样返回。
func LocationDisplayText(value string) string {
	switch value {
	case "location.intranet":
		return "内网 IP"
	case "location.unknown":
		return "未知位置"
	default:
		return value
	}
}

// InitLocationService 初始化地理位置服务 (空实现)
func InitLocationService() {
	// 地理位置服务尚未实现，预留接口供后续集成第三方 IP 地理库
}
