package main

import (
	"flag"
	"fmt"
	"log"

	browsercookie "github.com/g-lib/go-browser-cookie"
)

func main() {
	browserFlag := flag.String("browser", "edge", "浏览器类型: edge 或 chrome")
	domainFlag := flag.String("domain", "", "按域名过滤 cookie，例如 .example.com")
	flag.Parse()

	browser, err := browsercookie.ParseBrowserKind(*browserFlag)
	if err != nil {
		log.Fatal(err)
	}

	cookieString, cookieMap, cookies, err := browsercookie.GetCookies(browser, *domainFlag)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("\n\tCookies:")
	for _, c := range cookies {
		fmt.Printf("%s\t%s\t%s\n", c.Domain, c.Name, c.Value)
	}

	fmt.Println("\n\tCookie Map:")
	for k, v := range cookieMap {
		fmt.Printf("%s=%s\n", k, v)
	}

	fmt.Println("\n\tCookie String:")
	fmt.Println(cookieString)

}
