package app

import (
	"bytes"
	"embed"
	"fmt"
	"github.com/google/uuid"
	"html/template"
	"net"
	"os"
	"runtime"
	"time"
)

var c = make(chan os.Signal, 1)

var timeout = time.Second * 5

type MSG map[string]interface{}

type Captcha struct {
	Src  string
	X    int
	Y    int
	Tips string
}

type WebHook struct {
	Url    string
	Method string
	Key    string
}

// 格式化年月日
func FormatAsDate(t time.Time) string {
	year, month, day := t.Date()
	return fmt.Sprintf("%d%02d/%02d", year, month, day)
}

// 获取年份
func GetYear() string {
	t := time.Now()
	year, _, _ := t.Date()
	return fmt.Sprintf("%d", year)
}

// 获取当前年月日
func GetDate() string {
	t := time.Now()
	year, month, day := t.Date()
	return fmt.Sprintf("%d-%02d-%02d", year, month, day)
}

// 随机获取一个头像
func Getavator() string {
	Uuid := uuid.New().String()
	grav_url := "https://www.gravatar.com/avatar/" + Uuid
	return grav_url
}

type info struct {
	Root       string
	Version    string
	Hostname   string
	Interfaces interface{}
	Goarch     string
	Goos       string
	// VirtualMemory *mem.VirtualMemoryStat
	Sys         uint64
	CpuInfoStat struct {
		Count   int
		Percent []float64
	}
}

func GetServerInfo() *info {
	root := runtime.GOROOT()          // GO 路径
	version := runtime.Version()      // GO 版本信息
	hostname, _ := os.Hostname()      // 获得PC名
	interfaces, _ := net.Interfaces() // 获得网卡信息
	goarch := runtime.GOARCH          // 系统构架 386、amd64
	goos := runtime.GOOS              // 系统版本 windows
	Info := &info{
		Root:       root,
		Version:    version,
		Hostname:   hostname,
		Interfaces: interfaces,
		Goarch:     goarch,
		Goos:       goos,
	}

	// v, _ := mem.VirtualMemory()
	// Info.VirtualMemory = v
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	Info.Sys = ms.Sys
	// Info.CpuInfoStat.Count, _ = cpu.Counts(true)
	// Info.CpuInfoStat.Percent, _ = cpu.Percent(0, true)
	return Info
}

// 字节的单位转换 保留两位小数
func FormatFileSize(fileSize uint64) (size string) {
	if fileSize < 1024 {
		// return strconv.FormatInt(fileSize, 10) + "B"
		return fmt.Sprintf("%.2fB", float64(fileSize)/float64(1))
	} else if fileSize < (1024 * 1024) {
		return fmt.Sprintf("%.2fKB", float64(fileSize)/float64(1024))
	} else if fileSize < (1024 * 1024 * 1024) {
		return fmt.Sprintf("%.2fMB", float64(fileSize)/float64(1024*1024))
	} else if fileSize < (1024 * 1024 * 1024 * 1024) {
		return fmt.Sprintf("%.2fGB", float64(fileSize)/float64(1024*1024*1024))
	} else if fileSize < (1024 * 1024 * 1024 * 1024 * 1024) {
		return fmt.Sprintf("%.2fTB", float64(fileSize)/float64(1024*1024*1024*1024))
	} else { // if fileSize < (1024 * 1024 * 1024 * 1024 * 1024 * 1024)
		return fmt.Sprintf("%.2fEB", float64(fileSize)/float64(1024*1024*1024*1024*1024))
	}
}

func pickUnusedPort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	port := l.Addr().(*net.TCPAddr).Port
	if err := l.Close(); err != nil {
		return 0, err
	}
	return port, nil
}

// Msg 消息结构体
type Msg struct {
	Msg  string // 错误/成功 信息
	URL  string // 跳转地址
	Wait int64  // 跳转等待时间 秒
}

// HTMLFilesHandler 从路径读取模板
func HTMLFilesHandler(data Msg, files ...string) (template.HTML, error) {
	if data.Wait == 0 {
		data.Wait = 3
	}
	cbuf := new(bytes.Buffer)
	t, err := template.ParseFS(GetStaticFs(), files...)
	if err != nil {
		return "", err
	} else if err := t.Execute(cbuf, data); err != nil {
		return "", err
	}
	return template.HTML(cbuf.String()), err
}

// HTMLFilesHandlerString 从路径读取模板字符串
func HTMLFilesHandlerString(data Msg, files ...string) (string, error) {
	if data.Wait == 0 {
		data.Wait = 3
	}
	cbuf := new(bytes.Buffer)
	t, err := template.ParseFS(GetStaticFs(), files...)
	if err != nil {
		return "", err
	} else if err := t.Execute(cbuf, data); err != nil {
		return "", err
	}
	return cbuf.String(), err
}

var f embed.FS

// GetStaticFs 获取静态资源打包对象
func GetStaticFs() embed.FS {
	return f
}

// Asset loads and returns the asset for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func Asset(name string) ([]byte, error) {
	return f.ReadFile(name)
}

// AssetNames returns the names of the assets.
func AssetNames() []string {
	ds, _ := f.ReadDir("static/html")
	names := make([]string, 0, len(ds))
	for _, d := range ds {
		names = append(names, fmt.Sprintf("static/html/%s", d.Name()))
	}
	return names
}
