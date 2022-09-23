package app

import (
	"ck_selenium/util"
	"context"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/stitch-june/selenium"
	"github.com/stitch-june/selenium/firefox"
	"go.uber.org/dig"
	"os"
	"runtime"
	"time"
)

var geckoVersion = "v0.30.0"

var geckoMirrors = "https://npm.taobao.org/mirrors/geckodriver"

type GeckoDriver struct {
	Wd         selenium.WebDriver
	Service    *selenium.Service
	Ct         *dig.Container
	DriverPath string
}

func (ge *GeckoDriver) GetWd() selenium.WebDriver {
	if ge.Wd == nil {
		ge.Init()
	}
	if ge.Wd == nil {
		return nil
	}
	if _, err := ge.Wd.Status(); err != nil {
		ge.Wd.NewSession()
	}
	return ge.Wd
}

func (ge *GeckoDriver) GetService() *selenium.Service {
	return ge.Service
}

func (ge *GeckoDriver) GetFileDriverPath() string {
	return ge.DriverPath
}

func NewGeckoService(ct *dig.Container) SeInterface {
	ch := &GeckoDriver{Ct: ct}
	ch.Init()
	return ch
}

// 获取 系统和架构，读取geckodriver的位置
func (ge *GeckoDriver) GetDriverPath(ct *dig.Container) (string, error) {
	src := ""
	osname := ""
	filename := ""
	bfile := "geckodriver"
	var err error
	switch runtime.GOOS {
	case "windows":
		osname = "win"
		switch runtime.GOARCH {
		case "amd64":
			src = fmt.Sprintf("%s/%s/geckodriver-%s-%s64.zip", geckoMirrors, geckoVersion, geckoVersion, osname)
			filename = fmt.Sprintf("geckodriver-%s-%s64.zip", geckoVersion, osname)
			break
		case "386":
			src = fmt.Sprintf("%s/%s/geckodriver-%s-%s32.zip", geckoMirrors, geckoVersion, geckoVersion, osname)
			filename = fmt.Sprintf("geckodriver-%s-%s32.zip", geckoVersion, osname)
			break
		default:
			return "", errors.New("not support os")
		}
		bfile = "geckodriver.exe"
		break
	case "darwin":
		osname = "macos"
		if runtime.GOARCH == "arm64" {
			src = fmt.Sprintf("%s/%s/geckodriver-%s-%s-aarch64.tar.gz", geckoMirrors, geckoVersion, geckoVersion, osname)
			filename = fmt.Sprintf("geckodriver-%s-%s-aarch64.tar.gz", geckoVersion, osname)
		} else {
			src = fmt.Sprintf("%s/%s/geckodriver-%s-%s.tar.gz", geckoMirrors, geckoVersion, geckoVersion, osname)
			filename = fmt.Sprintf("geckodriver-%s-%s.tar.gz", geckoVersion, osname)
		}
		break
	case "linux":
		osname = "linux"
		switch runtime.GOARCH {
		case "amd64":
			src = fmt.Sprintf("%s/%s/geckodriver-%s-%s64.tar.gz", geckoMirrors, geckoVersion, geckoVersion, osname)
			filename = fmt.Sprintf("geckodriver-%s-%s64.tar.gz", geckoVersion, osname)
			break
		case "386":
			src = fmt.Sprintf("%s/%s/geckodriver-%s-%s32.tar.gz", geckoMirrors, geckoVersion, geckoVersion, osname)
			filename = fmt.Sprintf("geckodriver-%s-%s32.tar.gz", geckoVersion, osname)
			break
		default:
			return "", errors.New("not support os")
		}
		break
	default:
		log.Errorf("os =%s,arch=%s \n", runtime.GOOS, runtime.GOARCH)
		return "", errors.New("not support os")
	}
	switch runtime.GOARCH {
	case "arm64":
		if osname != "macos" {
			return "", errors.New("not support arch")
		}
		break
	case "amd64":
		break
	case "386":
		if osname != "win" {
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

func (ge *GeckoDriver) Init() (err error) {
	p, _ := pickUnusedPort()
	// p := 18777
	opts := []selenium.ServiceOption{
		// selenium.StartFrameBuffer(),           // Start an X frame buffer for the browser to run in.
		// selenium.Output(os.Stderr), // Output debug information to STDERR.
	}

	ge.DriverPath, err = ge.GetDriverPath(ge.Ct)
	if err != nil {
		return err
	}
	selenium.SetDebug(false)
	ge.Service, err = selenium.NewGeckoDriverService(ge.DriverPath, p, opts...)
	if err != nil {
		return err
	}

	// Connect to the WebDriver instance running locally.
	caps := selenium.Capabilities{"browserName": "firefox"}
	ge.Wd, err = selenium.NewRemote(caps, fmt.Sprintf("http://localhost:%d", p))
	if err != nil {
		return err
	}
	// firfox参数
	firefoxCaps := firefox.Capabilities{
		Binary: "",
		Args: []string{
			"--user-agent=Mozilla/5.0 (iPhone; CPU iPhone OS 14_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.0.3 Mobile/15E148 Safari/604.1",
			"--window-size=375,812",
		},
	}
	caps.AddFirefox(firefoxCaps)
	// 调整浏览器长宽高
	return ge.Wd.ResizeWindow("", 375, 812)
}
func (ge *GeckoDriver) SeRun() (err error) {

	ge.GetWd().DeleteAllCookies()
	// Navigate to the simple playground interface.
	if err = ge.GetWd().Get("https://home.m.jd.com/myJd/newhome.action"); err != nil {
		return err
	}
	// go ge.GetCookies(ct)
	return err
}

func (ge *GeckoDriver) CheckLastVersion() (version string, err error) {
	return geckoVersion, nil
}

func (ch *GeckoDriver) EnterPhone(phone string) error {
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
	check.Click()
	ele, err := ch.Wd.FindElement(selenium.ByXPATH, "//*[@id=\"app\"]/div/div[3]/p[1]/input")
	if err != nil {
		return err
	}
	err = ele.SendKeys(phone)
	return err
}

func (ch *GeckoDriver) SendSMS() error {
	if ch.Wd == nil {
		return errors.New("not init")
	}
	button, err := ch.Wd.FindElement(selenium.ByXPATH, "//*[@id=\"app\"]/div/div[3]/p[2]/button")
	if err != nil {
		return err
	}
	return button.Click()
}

func (ch *GeckoDriver) GetCaptcha() (*Captcha, error) {
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

func (ch *GeckoDriver) CheckCaptcha(mouseType string, x, y int) {
	switch mouseType {
	case "start":
		ch.Wd.StorePointerActions("touch1", selenium.MousePointer,
			selenium.PointerDownAction(selenium.LeftButton),
			selenium.PointerMoveAction(0, selenium.Point{X: x, Y: y}, selenium.FromViewport))
	case "end":
		ch.Wd.StorePointerActions("touch1", selenium.MousePointer,
			selenium.PointerMoveAction(0, selenium.Point{X: x, Y: y}, selenium.FromViewport),
			selenium.PointerUpAction(selenium.LeftButton))
	case "move":
		ch.Wd.StorePointerActions("touch1", selenium.MousePointer,
			selenium.PointerMoveAction(0, selenium.Point{X: x, Y: y}, selenium.FromPointer))
	}
}

func (ch *GeckoDriver) CheckCaptcha2(actions []selenium.PointerAction) error {
	if ch.Wd == nil {
		return errors.New("not init")
	}
	ch.Wd.StorePointerActions("touch1", selenium.MousePointer, actions...)
	err := ch.Wd.PerformActions()
	if err != nil {
		return err
	}
	err = ch.Wd.ReleaseActions()
	return err
}

func (ch *GeckoDriver) GetCookie() (string, error) {
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

func (ch *GeckoDriver) GetScreenShort() ([]byte, error) {
	if ch.Wd == nil {
		return nil, errors.New("not init")
	}
	return ch.Wd.Screenshot()
}

func (ch *GeckoDriver) GetSource() (string, error) {
	if ch.Wd == nil {
		return "", errors.New("not init")
	}
	return ch.Wd.PageSource()
}

func (ch *GeckoDriver) EnterSmsCode(smsCode string) error {
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

func (ch *GeckoDriver) ChangeLoginType() error {
	if ch.Wd == nil {
		return errors.New("not init")
	}
	check, err := ch.Wd.FindElement(selenium.ByXPATH, "//*[@id=\"app\"]/div/p[1]/span[1]")
	if err != nil {
		return err
	}
	return check.Click()
}

func (ch *GeckoDriver) EnterUserName(user string) error {
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

func (ch *GeckoDriver) EnterPasswd(passwd string) error {
	if ch.Wd == nil {
		return errors.New("not init")
	}
	ele, err := ch.Wd.FindElement(selenium.ByXPATH, "//*[@id=\"pwd\"]")
	if err != nil {
		return err
	}
	return ele.SendKeys(passwd)
}

func (ch *GeckoDriver) SubmitLogin() error {
	if ch.Wd == nil {
		return errors.New("not init")
	}
	sub, err := ch.Wd.FindElement(selenium.ByXPATH, "//*[@id=\"app\"]/div/a")
	if err != nil {
		return err
	}
	return sub.Click()
}

func (ch *GeckoDriver) Close() {
	ch.GetWd().Close()
}

func (ch *GeckoDriver) Quit() {
	ch.GetWd().Quit()
	ch.GetService().Stop()
	os.RemoveAll(ch.GetFileDriverPath())
}

func (ch *GeckoDriver) SecondSmsCheck() error {
	if ch.Wd == nil {
		return errors.New("not init")
	}
	check, err := ch.Wd.FindElement(selenium.ByXPATH, "//*[@id=\"app\"]/div/div[2]/div[2]/span/a/span")
	if err != nil {
		return err
	}
	return check.Click()
}

func (ch *GeckoDriver) SecondSmsSend() error {
	if ch.Wd == nil {
		return errors.New("not init")
	}
	btn, err := ch.Wd.FindElement(selenium.ByXPATH, "//*[@id=\"app\"]/div/div[2]/div[2]/button")
	if err != nil {
		return err
	}
	return btn.Click()
}

func (ch *GeckoDriver) EnterSecondSmsCode(code string) error {
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
