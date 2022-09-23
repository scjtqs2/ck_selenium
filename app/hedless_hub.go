package app

import (
	"embed"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/stitch-june/selenium"
	"github.com/stitch-june/selenium/chrome"
	"go.uber.org/dig"
	"io/fs"
	"net/http"
	"time"
)

type WdDriver struct {
	Wd selenium.WebDriver
	Ct *dig.Container
}

func (w *WdDriver) GetDriverPath(ct *dig.Container) (string, error) {
	return "", nil
}

func (w *WdDriver) Init() error {
	var err error
	var urlll string
	w.Ct.Invoke(func(addr string) {
		urlll = addr
	})
	var f embed.FS
	w.Ct.Invoke(func(static embed.FS) {
		f = static
	})
	selenium.HTTPClient = &http.Client{
		Timeout: time.Second * 10,
	}
	// chrome参数
	caps := selenium.Capabilities{"browserName": "chrome"}
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
		// ExcludeSwitches: []string{"enable-automation"},

		Args: []string{
			"--headless", // 设置Chrome无头模式，在linux下运行，需要设置这个参数，否则会报错
			// "--no-sandbox",
			"--user-agent=Mozilla/5.0 (iPhone; CPU iPhone OS 14_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.0.3 Mobile/15E148 Safari/604.1",
			"--window-size=375,812",
		},
		W3C: true,
	}
	caps.AddChrome(chromeCaps)

	// // firfox参数
	// caps := selenium.Capabilities{"browserName": "firefox"}
	// firefoxCaps := firefox.Capabilities{
	// 	Binary: "",
	// 	Args: []string{
	// 		"--user-agent=Mozilla/5.0 (iPhone; CPU iPhone OS 14_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.0.3 Mobile/15E148 Safari/604.1",
	// 		"--window-size=375,812",
	// 	},
	// }
	// caps.AddFirefox(firefoxCaps)
	// 调整浏览器长宽高

	w.Wd, err = selenium.NewRemote(caps, urlll)
	if err != nil {
		return err
	}
	stealthJs, err := fs.ReadFile(f, "static/js/stealth.min.js")
	if err != nil {
		return err
	}
	w.Wd.ExecuteChromeDPCommand("Page.addScriptToEvaluateOnNewDocument", map[string]string{
		"source": string(stealthJs),
	})
	return w.Wd.ResizeWindow("", 375, 812)
}
func (w *WdDriver) SeRun() error {
	var err error
	w.GetWd().DeleteAllCookies()
	if err = w.Wd.Get("https://home.m.jd.com/myJd/newhome.action"); err != nil {
		return err
	}

	return err
}

func (w *WdDriver) GetWd() selenium.WebDriver {
	if w.Wd == nil {
		w.Init()
	}

	if w.Wd == nil {
		return nil
	}
	if _, err := w.Wd.Status(); err != nil {
		w.Wd.NewSession()
	}
	return w.Wd
}

func (w *WdDriver) GetService() *selenium.Service {
	return nil
}

func (w *WdDriver) GetFileDriverPath() string {
	return ""
}

func (w *WdDriver) CheckLastVersion() (version string, err error) {
	return "", nil
}

func NewWdService(ct *dig.Container) SeInterface {
	ch := &WdDriver{
		Ct: ct,
	}
	ch.Init()
	return ch
}

func (ch *WdDriver) EnterPhone(phone string) error {
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

func (ch *WdDriver) SendSMS() error {
	if ch.Wd == nil {
		return errors.New("not init")
	}
	button, err := ch.Wd.FindElement(selenium.ByXPATH, "//*[@id=\"app\"]/div/div[3]/p[2]/button")
	if err != nil {
		return err
	}
	return button.Click()
}

func (ch *WdDriver) GetCaptcha() (*Captcha, error) {
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

func (ch *WdDriver) CheckCaptcha(mouseType string, x, y int) {
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

func (ch *WdDriver) CheckCaptcha2(actions []selenium.PointerAction) error {
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

func (ch *WdDriver) GetCookie() (string, error) {
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

func (ch *WdDriver) GetScreenShort() ([]byte, error) {
	if ch.Wd == nil {
		return nil, errors.New("not init")
	}
	return ch.Wd.Screenshot()
}

func (ch *WdDriver) GetSource() (string, error) {
	if ch.Wd == nil {
		return "", errors.New("not init")
	}
	return ch.Wd.PageSource()
}

func (ch *WdDriver) EnterSmsCode(smsCode string) error {
	input, err := ch.Wd.FindElement(selenium.ByXPATH, "//*[@id=\"authcode\"]")
	if err != nil {
		return err
	}
	err = input.SendKeys(smsCode)
	return err
}

func (ch *WdDriver) ChangeLoginType() error {
	if ch.Wd == nil {
		return errors.New("not init")
	}
	check, err := ch.Wd.FindElement(selenium.ByXPATH, "//*[@id=\"app\"]/div/p[1]/span[1]")
	if err != nil {
		return err
	}
	return check.Click()
}

func (ch *WdDriver) EnterUserName(user string) error {
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

func (ch *WdDriver) EnterPasswd(passwd string) error {
	if ch.Wd == nil {
		return errors.New("not init")
	}
	ele, err := ch.Wd.FindElement(selenium.ByXPATH, "//*[@id=\"pwd\"]")
	if err != nil {
		return err
	}
	return ele.SendKeys(passwd)
}

func (ch *WdDriver) SubmitLogin() error {
	if ch.Wd == nil {
		return errors.New("not init")
	}
	sub, err := ch.Wd.FindElement(selenium.ByXPATH, "//*[@id=\"app\"]/div/a")
	if err != nil {
		return err
	}
	return sub.Click()
}

func (ch *WdDriver) Close() {
	ch.Wd.Close()
}

func (ch *WdDriver) Quit() {
	if ch.Wd != nil {
		ch.Wd.Quit()
		ch.Wd = nil
	}
}

func (ch *WdDriver) SecondSmsCheck() error {
	if ch.Wd == nil {
		return errors.New("not init")
	}
	check, err := ch.Wd.FindElement(selenium.ByXPATH, "//*[@id=\"app\"]/div/div[2]/div[2]/span/a/span")
	if err != nil {
		return err
	}
	return check.Click()
}

func (ch *WdDriver) SecondSmsSend() error {
	if ch.Wd == nil {
		return errors.New("not init")
	}
	btn, err := ch.Wd.FindElement(selenium.ByXPATH, "//*[@id=\"app\"]/div/div[2]/div[2]/button")
	if err != nil {
		return err
	}
	return btn.Click()
}

func (ch *WdDriver) EnterSecondSmsCode(code string) error {
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
