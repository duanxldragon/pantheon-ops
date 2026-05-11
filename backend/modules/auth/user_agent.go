package auth

import "strings"

func parseClientInfo(userAgent string) ClientInfoResp {
	trimmed := strings.TrimSpace(userAgent)
	if trimmed == "" {
		return ClientInfoResp{
			Browser:   "Unknown",
			OS:        "Unknown",
			Device:    "Unknown",
			UserAgent: "",
		}
	}

	ua := strings.ToLower(trimmed)

	return ClientInfoResp{
		Browser:   detectBrowser(ua),
		OS:        detectOS(ua),
		Device:    detectDevice(ua),
		UserAgent: trimmed,
	}
}

func detectBrowser(ua string) string {
	switch {
	case strings.Contains(ua, "micromessenger"):
		return "WeChat"
	case strings.Contains(ua, "edg/"):
		return "Edge"
	case strings.Contains(ua, "opr/"), strings.Contains(ua, "opera"):
		return "Opera"
	case strings.Contains(ua, "chrome/") && !strings.Contains(ua, "edg/") && !strings.Contains(ua, "opr/"):
		return "Chrome"
	case strings.Contains(ua, "firefox/"):
		return "Firefox"
	case strings.Contains(ua, "safari/") && strings.Contains(ua, "version/") && !strings.Contains(ua, "chrome/"):
		return "Safari"
	default:
		return "Unknown"
	}
}

func detectOS(ua string) string {
	switch {
	case strings.Contains(ua, "windows"):
		return "Windows"
	case strings.Contains(ua, "mac os x"), strings.Contains(ua, "macintosh"):
		return "macOS"
	case strings.Contains(ua, "iphone"), strings.Contains(ua, "ipad"), strings.Contains(ua, "ios"):
		return "iOS"
	case strings.Contains(ua, "android"):
		return "Android"
	case strings.Contains(ua, "linux"), strings.Contains(ua, "x11"):
		return "Linux"
	default:
		return "Unknown"
	}
}

func detectDevice(ua string) string {
	switch {
	case strings.Contains(ua, "ipad"):
		return "iPad"
	case strings.Contains(ua, "iphone"):
		return "iPhone"
	case strings.Contains(ua, "android") && strings.Contains(ua, "mobile"):
		return "Android Phone"
	case strings.Contains(ua, "android"):
		return "Android Tablet"
	case strings.Contains(ua, "mobile"):
		return "Mobile"
	default:
		return "Desktop"
	}
}
