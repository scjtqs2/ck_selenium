package app

import (
	"ck_selenium/util"
	"context"
	"errors"
	"fmt"
	"github.com/guonaihong/gout"
	log "github.com/sirupsen/logrus"
	"github.com/stitch-june/selenium"
	"github.com/stitch-june/selenium/chrome"
	"go.uber.org/dig"
	"os"
	"runtime"
	"time"
)

var chromeVersion = "95.0.4638.17"

var chromeMirrors = "https://npm.taobao.org/mirrors/chromedriver"

type ChromeDriver struct {
	Wd         selenium.WebDriver
	Service    *selenium.Service
	Ct         *dig.Container
	DriverPath string
}

func (ch *ChromeDriver) GetWd() selenium.WebDriver {
	if ch.Wd == nil {
		ch.Init()
	}
	if ch.Wd == nil {
		return nil
	}
	if _, err := ch.Wd.Status(); err != nil {
		ch.Wd.NewSession()
	}
	return ch.Wd
}

func (ch *ChromeDriver) GetService() *selenium.Service {
	return ch.Service
}

func (ch *ChromeDriver) GetFileDriverPath() string {
	return ch.DriverPath
}

func NewChromeService(ct *dig.Container) SeInterface {
	ch := &ChromeDriver{Ct: ct}
	ch.Init()
	return ch
}

// 获取 系统和架构，读取geckodriver的位置
func (ch *ChromeDriver) GetDriverPath(ct *dig.Container) (string, error) {
	src := ""
	osname := ""
	filename := ""
	bfile := "chromedriver"
	var err error
	chromeVersion, err = ch.CheckLastVersion()
	switch runtime.GOOS {
	case "windows":
		osname = "win32"
		src = fmt.Sprintf("%s/%s/chromedriver_%s.zip", chromeMirrors, chromeVersion, osname)
		filename = fmt.Sprintf("chromedriver_%s.zip", osname)
		bfile = "chromedriver.exe"
		break
	case "darwin":
		osname = "mac64"
		if runtime.GOARCH == "arm64" {
			src = fmt.Sprintf("%s/%s/chromedriver_%s-m1.zip", chromeMirrors, chromeVersion, osname)
			filename = fmt.Sprintf("chromedriver_%s-m1.zip", osname)
		} else {
			src = fmt.Sprintf("%s/%s/chromedriver_%s.zip", chromeMirrors, chromeVersion, osname)
			filename = fmt.Sprintf("chromedriver_%s.zip", osname)
		}
		break
	case "linux":
		osname = "linux64"
		src = fmt.Sprintf("%s/%s/chromedriver_%s.zip", chromeMirrors, chromeVersion, osname)
		filename = fmt.Sprintf("chromedriver_%s.zip", osname)
		break
	default:
		log.Errorf("os =%s,arch=%s \n", runtime.GOOS, runtime.GOARCH)
		return "", errors.New("not support os")
	}
	switch runtime.GOARCH {
	case "arm64":
		if osname != "mac64" {
			return "", errors.New("not support arch")
		}
		break
	case "amd64":
		break
	case "386":
		if osname != "win32" {
			return "", errors.New("not support arch")
		}
		break
	default:
		return "", errors.New("not support arch")
	}
	dst := "./tmp"
	util.DownloadSingle(context.Background(), src, fmt.Sprintf("%s/%s", dst, filename))
	util.Unpack(context.Background(), fmt.Sprintf("%s/%s", dst, filename), dst)
	return fmt.Sprintf("%s/%s", dst, bfile), err
}

func (ch *ChromeDriver) Init() (err error) {
	p, _ := pickUnusedPort()
	// p := 18777
	opts := []selenium.ServiceOption{
		// selenium.StartFrameBuffer(),           // Start an X frame buffer for the browser to run in.
		// selenium.Output(os.Stderr), // Output debug information to STDERR.
	}
	ch.DriverPath, err = ch.GetDriverPath(ch.Ct)
	if err != nil {
		return err
	}
	selenium.SetDebug(false)
	ch.Service, err = selenium.NewChromeDriverService(ch.DriverPath, p, opts...)
	if err != nil {
		return err
	}

	// Connect to the WebDriver instance running locally.
	caps := selenium.Capabilities{"browserName": "chrome"}
	// chrome参数
	chromeCaps := chrome.Capabilities{
		Path: "",
		MobileEmulation: &chrome.MobileEmulation{
			// DeviceName: "iPhone X",
			DeviceMetrics: &chrome.DeviceMetrics{
				Width:  375,
				Height: 812,
			},
			UserAgent: "Mozilla/5.0 (iPhone; CPU iPhone OS 14_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.0.3 Mobile/15E148 Safari/604.1",
		},
		Args: []string{
			"--headless", // 设置Chrome无头模式，在linux下运行，需要设置这个参数，否则会报错
			// "--no-sandbox",
			"--user-agent=Mozilla/5.0 (iPhone; CPU iPhone OS 14_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.0.3 Mobile/15E148 Safari/604.1",
			"--window-size=375,812",
		},
	}
	caps.AddChrome(chromeCaps)
	ch.Wd, err = selenium.NewRemote(caps, fmt.Sprintf("http://localhost:%d/wd/hub", p))
	if err != nil {
		return err
	}
	// 调整浏览器长宽高
	return ch.Wd.ResizeWindow("", 375, 812)
}
func (ch *ChromeDriver) SeRun() (err error) {
	ch.GetWd().DeleteAllCookies()
	// Navigate to the simple playground interface.
	if err = ch.GetWd().Get("https://home.m.jd.com/myJd/newhome.action"); err != nil {
		return err
	}
	// go ch.GetCookies(ct)
	return err
}

func (ch *ChromeDriver) CheckLastVersion() (version string, err error) {
	url := "https://npm.taobao.org/mirrors/chromedriver/LATEST_RELEASE"
	code := 0
	err = gout.GET(url).BindBody(&version).Code(&code).
		SetTimeout(timeout).
		F().Retry().Attempt(5).
		WaitTime(time.Millisecond * 500).MaxWaitTime(time.Second * 5).
		Do()
	if err != nil || code != 200 {
		return chromeVersion, err
	}
	log.Infof("latest chrome version =%s ", version)
	return version, err
}

func (ch *ChromeDriver) EnterPhone(phone string) error {
	if ch.Wd == nil {
		return errors.New("not init")
	}
	err := ch.Wd.WaitWithTimeout(func(wd selenium.WebDriver) (bool, error) {
		_, err := ch.Wd.FindElement(selenium.ByXPATH, "//*[@id=\"app\"]/div/p[2]/input")
		if err != nil {
			return false, err
		}
		return true, err
	}, 10*time.Second)
	check, err := ch.Wd.FindElement(selenium.ByXPATH, "//*[@id=\"app\"]/div/p[2]/input")
	if err != nil {
		return err
	}
	if is, _ := check.IsSelected(); !is {
		check.Click()
	}
	ele, err := ch.Wd.FindElement(selenium.ByXPATH, "//*[@id=\"app\"]/div/div[3]/p[1]/input")
	if err != nil {
		return err
	}
	err = ele.SendKeys(phone)
	return err
}

func (ch *ChromeDriver) EnterSmsCode(smsCode string) error {
	if ch.Wd == nil {
		return errors.New("not init")
	}
	input, err := ch.Wd.FindElement(selenium.ByXPATH, "//*[@id=\"authcode\"]")
	if err != nil {
		return err
	}
	err = input.SendKeys(smsCode)
	return err
}

func (ch *ChromeDriver) SendSMS() error {
	if ch.Wd == nil {
		return errors.New("not init")
	}
	button, err := ch.Wd.FindElement(selenium.ByXPATH, "//*[@id=\"app\"]/div/div[3]/p[2]/button")
	if err != nil {
		return err
	}
	return button.Click()
}

func (ch *ChromeDriver) GetCaptcha() (*Captcha, error) {
	if ch.Wd == nil {
		return &Captcha{}, errors.New("not init")
	}
	err := ch.Wd.WaitWithTimeout(func(wd selenium.WebDriver) (bool, error) {
		_, err := wd.FindElement(selenium.ByXPATH, "//*[@id=\"captcha_dom\"]")
		if err != nil {
			return false, err
		}
		return true, err
	}, 10*time.Second)
	img, err := ch.Wd.FindElement(selenium.ByXPATH, "//*[@id=\"cpc_img\"]")
	if err != nil {
		img, err = ch.Wd.FindElement(selenium.ByXPATH, "//*[@id=\"captcha_modal\"]/div/div[2]/img")
		if err != nil {
			return nil, err
		}
	}
	imgSrc, err := img.GetAttribute("src")
	if err != nil {
		return nil, err
	}
	point, err := img.Location()
	if err != nil {
		return nil, err
	}
	Tips, err := ch.Wd.FindElement(selenium.ByXPATH, "//*[@id=\"captcha_modal\"]/div/div[3]/div")
	if err != nil {
		return nil, err
	}
	tips, _ := Tips.Text()
	return &Captcha{
		Src:  imgSrc,
		X:    point.X,
		Y:    point.Y,
		Tips: tips,
	}, nil
}

func (ch *ChromeDriver) CheckCaptcha(mouseType string, x, y int) {
	switch mouseType {
	case "start":
		ch.Wd.StorePointerActions("touch1", selenium.MousePointer,
			selenium.PointerDownAction(selenium.LeftButton),
			selenium.PointerPauseAction(2500),
			selenium.PointerMoveAction(0, selenium.Point{X: x, Y: y}, selenium.FromViewport))
	case "end":
		ch.Wd.StorePointerActions("touch1", selenium.MousePointer,
			selenium.PointerMoveAction(0, selenium.Point{X: x, Y: y}, selenium.FromViewport),
			selenium.PointerPauseAction(2500),
			selenium.PointerUpAction(selenium.LeftButton))
		ch.Wd.PerformActions()
		ch.Wd.ReleaseActions()
	case "move":
		ch.Wd.StorePointerActions("touch1", selenium.MousePointer,
			selenium.PointerPauseAction(2500),
			selenium.PointerMoveAction(0, selenium.Point{X: x, Y: y}, selenium.FromPointer))
	}
}

func (ch *ChromeDriver) CheckCaptcha2(actions []selenium.PointerAction) error {
	if ch.Wd == nil {
		return errors.New("not init")
	}
	ch.Wd.StorePointerActions("touch1", selenium.MousePointer, actions...)
	err := ch.Wd.PerformActions()
	if err != nil {
		return err
	}
	ch.Wd.ReleaseActions()
	return err
}

func (ch *ChromeDriver) GetCookie() (string, error) {
	if ch.Wd == nil {
		return "", errors.New("not init")
	}
	cks, err := ch.Wd.GetCookies()
	var pt_pin, pt_key string
	if err != nil {
		return "", err
	}
	for _, v := range cks {
		if v.Name == "pt_pin" {
			pt_pin = v.Value
		}
		if v.Name == "pt_key" {
			pt_key = v.Value
		}
	}
	if pt_pin != "" && pt_key != "" {
		log.Info("############  登录成功，获取到 Cookie  #############")
		log.Infof("cookie=pt_pin=%s; pt_key=%s", pt_pin, pt_key)
		log.Info("####################################################")
		cookie := fmt.Sprintf("pt_pin=%s;pt_key=%s;", pt_pin, pt_key)
		return cookie, nil
	}
	return "", errors.New("empty cookie")
}

func (ch *ChromeDriver) GetScreenShort() ([]byte, error) {
	if ch.Wd == nil {
		return nil, errors.New("not init")
	}
	return ch.Wd.Screenshot()
}

func (ch *ChromeDriver) GetSource() (string, error) {
	if ch.Wd == nil {
		return "", errors.New("not init")
	}
	return ch.Wd.PageSource()
}

func (ch *ChromeDriver) ChangeLoginType() error {
	if ch.Wd == nil {
		return errors.New("not init")
	}
	check, err := ch.Wd.FindElement(selenium.ByXPATH, "//*[@id=\"app\"]/div/p[1]/span[1]")
	if err != nil {
		return err
	}
	return check.Click()
}

func (ch *ChromeDriver) EnterUserName(user string) error {
	if ch.Wd == nil {
		return errors.New("not init")
	}
	check, err := ch.Wd.FindElement(selenium.ByXPATH, "//*[@id=\"app\"]/div/p[2]/input")
	if err != nil {
		return err
	}
	if is, _ := check.IsSelected(); !is {
		check.Click()
	}
	ele, err := ch.Wd.FindElement(selenium.ByXPATH, "//*[@id=\"username\"]")
	if err != nil {
		return err
	}
	return ele.SendKeys(user)
}

func (ch *ChromeDriver) EnterPasswd(passwd string) error {
	if ch.Wd == nil {
		return errors.New("not init")
	}
	ele, err := ch.Wd.FindElement(selenium.ByXPATH, "//*[@id=\"pwd\"]")
	if err != nil {
		return err
	}
	return ele.SendKeys(passwd)
}

func (ch *ChromeDriver) SubmitLogin() error {
	if ch.Wd == nil {
		return errors.New("not init")
	}
	sub, err := ch.Wd.FindElement(selenium.ByXPATH, "//*[@id=\"app\"]/div/a")
	if err != nil {
		return err
	}
	return sub.Click()
}

func (ch *ChromeDriver) Close() {
	ch.GetWd().Close()
}

func (ch *ChromeDriver) Quit() {
	ch.GetWd().Quit()
	ch.GetService().Stop()
	os.RemoveAll(ch.GetFileDriverPath())
}
func (ch *ChromeDriver) SecondSmsCheck() error {
	if ch.Wd == nil {
		return errors.New("not init")
	}
	check, err := ch.Wd.FindElement(selenium.ByXPATH, "//*[@id=\"app\"]/div/div[2]/div[2]/span/a/span")
	if err != nil {
		return err
	}
	return check.Click()
}

func (ch *ChromeDriver) SecondSmsSend() error {
	if ch.Wd == nil {
		return errors.New("not init")
	}
	btn, err := ch.Wd.FindElement(selenium.ByXPATH, "//*[@id=\"app\"]/div/div[2]/div[2]/button")
	if err != nil {
		return err
	}
	return btn.Click()
}

func (ch *ChromeDriver) EnterSecondSmsCode(code string) error {
	if ch.Wd == nil {
		return errors.New("not init")
	}
	input, err := ch.Wd.FindElement(selenium.ByXPATH, "//*[@id=\"app\"]/div/div[2]/div[2]/div/input")
	if err != nil {
		return err
	}
	err = input.SendKeys(code)
	if err != nil {
		return err
	}
	btn, err := ch.Wd.FindElement(selenium.ByXPATH, "//*[@id=\"app\"]/div/div[2]/a[1]")
	if err != nil {
		return err
	}
	return btn.Click()
}
