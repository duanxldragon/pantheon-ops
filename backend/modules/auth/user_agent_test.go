package auth

import (
	"strings"
	"testing"
)

func TestParseClientInfo_Empty(t *testing.T) {
	info := parseClientInfo("")
	if info.Browser != "Unknown" || info.OS != "Unknown" || info.Device != "Unknown" {
		t.Fatalf("expected all Unknown, got Browser=%s OS=%s Device=%s", info.Browser, info.OS, info.Device)
	}
}

func TestParseClientInfo_Whitespace(t *testing.T) {
	info := parseClientInfo("   ")
	if info.Browser != "Unknown" || info.OS != "Unknown" || info.Device != "Unknown" {
		t.Fatal("expected all Unknown for whitespace")
	}
}

func TestDetectBrowser_Chrome(t *testing.T) {
	ua := "mozilla/5.0 chrome/120.0.0.0 safari/537.36"
	browser := detectBrowser(ua)
	if browser != "Chrome" {
		t.Fatalf("expected Chrome, got %s", browser)
	}
}

func TestDetectBrowser_Firefox(t *testing.T) {
	ua := "mozilla/5.0 firefox/121.0"
	browser := detectBrowser(ua)
	if browser != "Firefox" {
		t.Fatalf("expected Firefox, got %s", browser)
	}
}

func TestDetectBrowser_Edge(t *testing.T) {
	ua := "mozilla/5.0 chrome/120.0 edg/120.0.0.0 safari/537.36"
	browser := detectBrowser(ua)
	if browser != "Edge" {
		t.Fatalf("expected Edge, got %s", browser)
	}
}

func TestDetectBrowser_Safari(t *testing.T) {
	ua := "mozilla/5.0 macintosh safari/605.1.15 version/17.1"
	browser := detectBrowser(ua)
	if browser != "Safari" {
		t.Fatalf("expected Safari, got %s", browser)
	}
}

func TestDetectBrowser_SafariWithoutVersion(t *testing.T) {
	// Safari requires both "safari/" and "version/"; without "version/" it's Unknown
	ua := "mozilla/5.0 safari/605.1.15"
	browser := detectBrowser(ua)
	if browser != "Unknown" {
		t.Fatalf("expected Unknown (no version/), got %s", browser)
	}
}

func TestDetectBrowser_WeChat(t *testing.T) {
	ua := "mozilla/5.0 micromessenger/8.0.0"
	browser := detectBrowser(ua)
	if browser != "WeChat" {
		t.Fatalf("expected WeChat, got %s", browser)
	}
}

func TestDetectBrowser_Opera(t *testing.T) {
	ua := "mozilla/5.0 opr/100.0.0.0"
	browser := detectBrowser(ua)
	if browser != "Opera" {
		t.Fatalf("expected Opera, got %s", browser)
	}
}

func TestDetectBrowser_Unknown(t *testing.T) {
	ua := "somerandom/1.0"
	browser := detectBrowser(ua)
	if browser != "Unknown" {
		t.Fatalf("expected Unknown, got %s", browser)
	}
}

func TestDetectBrowser_ChromeTakesPrecedenceOverSafari(t *testing.T) {
	ua := "mozilla/5.0 chrome/120.0 safari/605.1 version/17.1"
	browser := detectBrowser(ua)
	if browser != "Chrome" {
		t.Fatalf("expected Chrome over Safari, got %s", browser)
	}
}

func TestDetectOS_Windows(t *testing.T) {
	ua := "mozilla/5.0 windows nt 10.0"
	os := detectOS(ua)
	if os != "Windows" {
		t.Fatalf("expected Windows, got %s", os)
	}
}

func TestDetectOS_macOS(t *testing.T) {
	ua := "mozilla/5.0 mac os x 10_15_7"
	os := detectOS(ua)
	if os != "macOS" {
		t.Fatalf("expected macOS, got %s", os)
	}
}

func TestDetectOS_Macintosh(t *testing.T) {
	ua := "mozilla/5.0 macintosh intel"
	os := detectOS(ua)
	if os != "macOS" {
		t.Fatalf("expected macOS for Macintosh, got %s", os)
	}
}

func TestDetectOS_iOS(t *testing.T) {
	ua := "mozilla/5.0 iphone ios 17"
	os := detectOS(ua)
	if os != "iOS" {
		t.Fatalf("expected iOS, got %s", os)
	}
}

func TestDetectOS_iPad(t *testing.T) {
	ua := "mozilla/5.0 ipad"
	os := detectOS(ua)
	if os != "iOS" {
		t.Fatalf("expected iOS for iPad, got %s", os)
	}
}

func TestDetectOS_Android(t *testing.T) {
	ua := "mozilla/5.0 android 14"
	os := detectOS(ua)
	if os != "Android" {
		t.Fatalf("expected Android, got %s", os)
	}
}

func TestDetectOS_Linux(t *testing.T) {
	ua := "mozilla/5.0 x11 linux x86_64"
	os := detectOS(ua)
	if os != "Linux" {
		t.Fatalf("expected Linux, got %s", os)
	}
}

func TestDetectOS_Unknown(t *testing.T) {
	ua := "someos/1.0"
	os := detectOS(ua)
	if os != "Unknown" {
		t.Fatalf("expected Unknown, got %s", os)
	}
}

func TestDetectDevice_Desktop(t *testing.T) {
	ua := "mozilla/5.0 windows nt 10.0"
	device := detectDevice(ua)
	if device != "Desktop" {
		t.Fatalf("expected Desktop, got %s", device)
	}
}

func TestDetectDevice_iPhone(t *testing.T) {
	ua := "mozilla/5.0 iphone"
	device := detectDevice(ua)
	if device != "iPhone" {
		t.Fatalf("expected iPhone, got %s", device)
	}
}

func TestDetectDevice_iPad(t *testing.T) {
	ua := "mozilla/5.0 ipad"
	device := detectDevice(ua)
	if device != "iPad" {
		t.Fatalf("expected iPad, got %s", device)
	}
}

func TestDetectDevice_AndroidPhone(t *testing.T) {
	ua := "mozilla/5.0 android mobile"
	device := detectDevice(ua)
	if device != "Android Phone" {
		t.Fatalf("expected Android Phone, got %s", device)
	}
}

func TestDetectDevice_AndroidTablet(t *testing.T) {
	ua := "mozilla/5.0 android tablet"
	device := detectDevice(ua)
	if device != "Android Tablet" {
		t.Fatalf("expected Android Tablet, got %s", device)
	}
}

func TestDetectDevice_Mobile(t *testing.T) {
	ua := "mozilla/5.0 mobile something"
	device := detectDevice(ua)
	if device != "Mobile" {
		t.Fatalf("expected Mobile, got %s", device)
	}
}

func TestParseClientInfo_ChromeOnWindows(t *testing.T) {
	ua := "Mozilla/5.0 Windows NT 10.0 Chrome/120.0 Safari/537.36"
	info := parseClientInfo(ua)
	if info.Browser != "Chrome" {
		t.Fatalf("expected Chrome, got %s", info.Browser)
	}
	if info.OS != "Windows" {
		t.Fatalf("expected Windows, got %s", info.OS)
	}
	if info.Device != "Desktop" {
		t.Fatalf("expected Desktop, got %s", info.Device)
	}
}

func TestParseClientInfo_iPhone(t *testing.T) {
	ua := "Mozilla/5.0 iPhone Safari/605.1 Version/17.1"
	info := parseClientInfo(ua)
	if info.OS != "iOS" {
		t.Fatalf("expected iOS, got %s", info.OS)
	}
	if info.Device != "iPhone" {
		t.Fatalf("expected iPhone, got %s", info.Device)
	}
}

func TestParseClientInfo_AndroidPhoneBrowser(t *testing.T) {
	ua := "Mozilla/5.0 Linux Android 14 Chrome/120.0 Mobile Safari/537.36"
	info := parseClientInfo(ua)
	if info.OS != "Android" {
		t.Fatalf("expected Android, got %s", info.OS)
	}
	if info.Device != "Android Phone" {
		t.Fatalf("expected Android Phone, got %s", info.Device)
	}
}

func TestParseClientInfo_ReturnsOriginalUA(t *testing.T) {
	ua := "Mozilla/5.0 Windows"
	info := parseClientInfo(ua)
	if info.UserAgent != ua {
		t.Fatalf("expected original UA preserved, got %q", info.UserAgent)
	}
}

func TestDetectBrowser_EdgeOverChrome(t *testing.T) {
	// "edg/" should match before "chrome/"
	ua := strings.ToLower("Mozilla/5.0 Chrome/120.0 Edg/120.0 Safari/537.36")
	browser := detectBrowser(ua)
	if browser != "Edge" {
		t.Fatalf("expected Edge over Chrome, got %s", browser)
	}
}

func TestDetectBrowser_OperaOverChrome(t *testing.T) {
	ua := strings.ToLower("Mozilla/5.0 Chrome/120.0 OPR/100.0 Safari/537.36")
	browser := detectBrowser(ua)
	if browser != "Opera" {
		t.Fatalf("expected Opera over Chrome, got %s", browser)
	}
}
