package browsercookie

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/storage"
	"github.com/chromedp/chromedp"
	"golang.org/x/sys/windows/registry"
)

type BrowserKind string

const (
	BrowserEdge   BrowserKind = "edge"
	BrowserChrome BrowserKind = "chrome"
)

func ParseBrowserKind(v string) (BrowserKind, error) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case string(BrowserEdge):
		return BrowserEdge, nil
	case string(BrowserChrome):
		return BrowserChrome, nil
	default:
		return "", fmt.Errorf("不支持的浏览器类型: %s", v)
	}
}

func resolveBrowserExecutable(kind BrowserKind) (string, error) {
	if runtime.GOOS != "windows" {
		return "", errors.New("仅支持 windows")
	}

	var candidates []string
	switch kind {
	case BrowserEdge:
		candidates = []string{
			filepath.Join(os.Getenv("ProgramFiles(x86)"), "Microsoft", "Edge", "Application", "msedge.exe"),
			filepath.Join(os.Getenv("ProgramFiles"), "Microsoft", "Edge", "Application", "msedge.exe"),
		}
	case BrowserChrome:
		candidates = []string{
			filepath.Join(os.Getenv("ProgramFiles"), "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(os.Getenv("ProgramFiles(x86)"), "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(os.Getenv("LocalAppData"), "Google", "Chrome", "Application", "chrome.exe"),
		}
	default:
		return "", fmt.Errorf("不支持的浏览器类型: %s", kind)
	}

	for _, p := range candidates {
		if p == "" {
			continue
		}
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	return "", fmt.Errorf("未找到 %s 可执行文件", kind)
}

func readUserDataDirFromRegistry(keyPath string) string {
	k, err := registry.OpenKey(registry.CURRENT_USER, keyPath, registry.QUERY_VALUE)
	if err != nil {
		return ""
	}
	defer k.Close()

	path, _, err := k.GetStringValue("UserDataDir")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(path)
}

func resolveUserDataDir(kind BrowserKind) (string, error) {
	if runtime.GOOS != "windows" {
		return "", errors.New("仅支持 windows")
	}

	var registryKey string
	var defaultPath string
	switch kind {
	case BrowserEdge:
		registryKey = `Software\Microsoft\Edge`
		defaultPath = filepath.Join(os.Getenv("LocalAppData"), "Microsoft", "Edge", "User Data")
	case BrowserChrome:
		registryKey = `Software\Google\Chrome`
		defaultPath = filepath.Join(os.Getenv("LocalAppData"), "Google", "Chrome", "User Data")
	default:
		return "", fmt.Errorf("不支持的浏览器类型: %s", kind)
	}

	if v := readUserDataDirFromRegistry(registryKey); v != "" {
		return v, nil
	}
	if defaultPath != "" {
		return defaultPath, nil
	}

	return "", fmt.Errorf("未找到 %s user data dir", kind)
}

func killBrowser(kind BrowserKind) {
	imageName := "msedge.exe"
	if kind == BrowserChrome {
		imageName = "chrome.exe"
	}
	exec.Command("taskkill", "/F", "/IM", imageName).Run()
}

func startBrowser(kind BrowserKind) (*exec.Cmd, error) {
	execPath, err := resolveBrowserExecutable(kind)
	if err != nil {
		return nil, err
	}
	userDataDir, err := resolveUserDataDir(kind)
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(
		execPath,
		"--remote-debugging-port=9222",
		"--user-data-dir="+userDataDir,
		"--restore-last-session",
		"--headless",
	)

	err = cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("启动 %s 失败: %w", kind, err)
	}

	return cmd, nil
}

func normalizeDomain(v string) string {
	return strings.TrimPrefix(strings.ToLower(strings.TrimSpace(v)), ".")
}

func domainMatch(cookieDomain, targetDomain string) bool {
	if targetDomain == "" {
		return true
	}
	c := normalizeDomain(cookieDomain)
	t := normalizeDomain(targetDomain)
	// Cookie domain 规则：target host == cookie domain，或 target host 是其子域名。
	return strings.Contains(c, t)
}

func buildCookieOutputs(cookies []*network.Cookie) (string, map[string]string) {
	cookieMap := make(map[string]string, len(cookies))
	for _, c := range cookies {
		cookieMap[c.Name] = c.Value
	}

	keys := make([]string, 0, len(cookieMap))
	for k := range cookieMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", k, cookieMap[k]))
	}
	return strings.Join(parts, "; "), cookieMap
}

func GetCookies(browser BrowserKind, domain string) (string, map[string]string, []*network.Cookie, error) {
	// 1) 关闭所有浏览器进程
	killBrowser(browser)
	time.Sleep(2 * time.Second)

	// 2) 启动浏览器并开启 remote debug
	cmd, err := startBrowser(browser)
	if err != nil {
		return "", nil, nil, err
	}
	defer func() {
		killBrowser(browser)
		if cmd != nil && cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	}()

	// 等待浏览器启动
	time.Sleep(3 * time.Second)

	// 3) 连接并获取 Cookies
	allocCtx, cancel := chromedp.NewRemoteAllocator(
		context.Background(),
		"http://127.0.0.1:9222",
	)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	var cookies []*network.Cookie

	err = chromedp.Run(ctx,
		network.Enable(),

		// 👉 建议访问一个站点，确保 cookie 被加载
		// chromedp.Navigate("https://www.baidu.com"),
		// chromedp.Sleep(2*time.Second),

		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			cookies, err = storage.GetCookies().Do(ctx)
			return err
		}),
	)

	if err != nil {
		return "", nil, nil, fmt.Errorf("获取 cookie 失败: %w", err)
	}

	filtered := make([]*network.Cookie, 0, len(cookies))
	for _, c := range cookies {
		if domainMatch(c.Domain, domain) {
			filtered = append(filtered, c)
		}
	}

	cookieString, cookieMap := buildCookieOutputs(filtered)
	return cookieString, cookieMap, filtered, nil
}
