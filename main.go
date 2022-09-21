package main

import (
	"ck_selenium/app"
	"embed"
	"go.uber.org/dig"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// 使用 go 1.16的新特性，自带的打包静态资源的包。
//
//go:embed static/*
var f embed.FS

var c = make(chan os.Signal, 1)

func main() {
	container := dig.New()
	container.Provide(func() (static embed.FS) {
		return f
	})
	container.Provide(func() (addr string) {
		return "http://127.0.0.1:4444/wd/hub"
	})
	http := app.NewHttp(container)
	// http.Se = app.NewChromeService(container)
	http.Se = app.NewWdService(container)
	defer func() {
		http.Se.GetWd().Quit()
		if http.Se.GetService() != nil {
			http.Se.GetService().Stop()
		}
		if http.Se.GetFileDriverPath() != "" {
			os.RemoveAll(http.Se.GetFileDriverPath())
		}
	}()
	err := http.Se.SeRun(container)
	if err != nil {
		panic(err)
	}
	time.Sleep(2 * time.Second)
	err = http.Se.EnterPhone("18612127452")
	if err != nil {
		panic(err)
	}
	time.Sleep(2 * time.Second)
	http.Se.SendSMS()

	signal.Notify(c, syscall.SIGINT, syscall.SIGKILL)
	<-c
}
