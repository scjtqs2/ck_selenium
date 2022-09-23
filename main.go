package main

import (
	"ck_selenium/app"
	"embed"
	"go.uber.org/dig"
	"os"
	"os/signal"
	"syscall"
)

// 使用 go 1.16的新特性，自带的打包静态资源的包。
//
//go:embed static/*
var f embed.FS
var Version string
var c = make(chan os.Signal, 1)

func main() {
	container := dig.New()
	container.Provide(func() (static embed.FS) {
		return f
	})
	container.Provide(func() (addr string) {
		add := os.Getenv("SELENIUM_CHROME_ADDR") // docker selenium hub 地址
		if add == "" {
			add = "http://127.0.0.1:4444/wd/hub" // docker selenium hub 地址
		}
		return add
	})
	container.Provide(func() app.WebHook {
		return app.WebHook{
			Url:    "https://jd.900109.xyz:8443/notify",
			Method: "GET",
			Key:    "hhkb",
		}
	})
	_ = app.NewHttp(container)
	// http.Se = app.NewChromeService(container) // for local Chrome
	// http.Se = app.NewWdService(container) // for docker
	// 阻塞主进程
	signal.Notify(c, syscall.SIGINT, syscall.SIGKILL)
	<-c
}
