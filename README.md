# go-browser-cookie

> Windows 下自动获取 Edge/Chrome Cookie 的 Go 工具库，支持 CLI 与包调用。

[![Go Version](https://img.shields.io/badge/Go-1.26%2B-00ADD8?logo=go)](https://go.dev/)
[![Platform](https://img.shields.io/badge/platform-Windows-blue)](#)
[![License](https://img.shields.io/badge/license-GPL-3.0-green)](./LICENSE)

自动完成：

- 浏览器可执行路径探测（Edge / Chrome）
- User Data Dir 探测（注册表优先，默认目录回退）
- Domain 过滤与多种结果输出

## 目录

- [特性](#特性)
- [快速开始](#快速开始)
- [CLI 用法](#cli-用法)
- [库用法](#库用法)
- [API](#api)
- [Domain 匹配规则](#domain-匹配规则)
- [注意事项](#注意事项)
- [License](#license)

## 特性

- 仅支持 Windows
- 支持浏览器：`edge`、`chrome`
- `GetCookies` 返回三类结果：
  - Cookie 字符串：`k1=v1; k2=v2`
  - `map[string]string`
  - 原始 `[]*network.Cookie`

## 快速开始

### 安装依赖（库模式）

```bash
go get github.com/g-lib/go-browser-cookie
```

### 直接运行 CLI

```bash
go run ./cli/go-browser-cookie -browser=edge -domain=example.com
```

## CLI 用法

```bash
go run ./cli/go-browser-cookie -browser=edge -domain=example.com
```

参数：

| 参数 | 默认值 | 说明 |
| --- | --- | --- |
| `-browser` | `edge` | 浏览器类型：`edge` 或 `chrome` |
| `-domain` | 空 | 过滤域名，可传 `example.com` 或 `.example.com` |

## 库用法

```go
package main

import (
	"fmt"
	"log"

	browsercookie "github.com/g-lib/go-browser-cookie"
)

func main() {
	browser, err := browsercookie.ParseBrowserKind("edge")
	if err != nil {
		log.Fatal(err)
	}

	cookieString, cookieMap, rawCookies, err := browsercookie.GetCookies(browser, "example.com")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("cookie string:", cookieString)
	fmt.Println("cookie map:", cookieMap)
	fmt.Println("raw cookie count:", len(rawCookies))
}
```

## API

### `ParseBrowserKind(v string) (BrowserKind, error)`

将字符串解析为浏览器类型，支持值：

- `edge`
- `chrome`

### `GetCookies(browser BrowserKind, domain string) (string, map[string]string, []*network.Cookie, error)`

返回值依次为：

1. Cookie 字符串（`k1=v1; k2=v2`）
2. `map[string]string` Cookie 数据
3. 过滤后的原始 Cookie 列表（`[]*network.Cookie`）
4. `error`

## Domain 匹配规则

当 `domain` 不为空时，匹配逻辑为：

- `target == cookieDomain`
- `target` 是 `cookieDomain` 的子域名

示例：

- `domain=example.com` 匹配 `example.com`、`.example.com`
- `domain=a.example.com` 匹配 `example.com`、`.example.com`、`a.example.com`

## 注意事项

- 读取前后会结束对应浏览器进程（`taskkill /F`）。
- 不建议直接用于用户正在操作的生产浏览器会话。
- 建议在独立用户目录或自动化环境中使用。

## License

MIT，见 [`LICENSE`](./LICENSE)。
