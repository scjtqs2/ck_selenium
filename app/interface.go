package app

import (
	"github.com/stitch-june/selenium"
	"go.uber.org/dig"
)

type SeInterface interface {
	GetDriverPath(ct *dig.Container) (string, error)
	Init() (err error)
	SeRun() error
	GetWd() selenium.WebDriver
	GetService() *selenium.Service
	GetFileDriverPath() string
	CheckLastVersion() (version string, err error)
	EnterPhone(phone string) error
	EnterSmsCode(smsCode string) error
	SendSMS() error
	GetCaptcha() (*Captcha, error)
	// CheckCaptcha(mouseType string, x, y int)
	CheckCaptcha2(actions []selenium.PointerAction) error
	GetCookie() (string, error)
	GetScreenShort() ([]byte, error)
	GetSource() (string, error)
	ChangeLoginType() error
	EnterUserName(user string) error
	EnterPasswd(passwd string) error
	SubmitLogin() error
	Close()
	Quit()
	SecondSmsCheck() error
	SecondSmsSend() error
	EnterSecondSmsCode(code string) error
}

var SeType string

func NewSeService(ct *dig.Container) (SeInterface, error) {
	switch SeType {
	case "firefox":
		return NewGeckoService(ct), nil
	case "chrome":
		return NewChromeService(ct), nil
	default:
		return NewWdService(ct), nil
	}
}
