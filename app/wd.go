/**
 * Copyright (c) 2022 Oray Inc. All rights reserved.
 *
 * No Part of this file may be reproduced, stored
 * in a retrieval system, or transmitted, in any form, or by any means,
 * electronic, mechanical, photocopying, recording, or otherwise,
 * without the prior consent of Oray Inc.
 *
 *
 * @author qiushi
 */
package app

import (
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/tebeka/selenium"
	"github.com/tebeka/selenium/chrome"
	"go.uber.org/dig"
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

func (w *WdDriver) SeRun(ct *dig.Container) error {
	var err error
	var urlll string
	ct.Invoke(func(addr string) {
		urlll = addr
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
		Args: []string{
			"--headless", // 设置Chrome无头模式，在linux下运行，需要设置这个参数，否则会报错
			// "--no-sandbox",
			"--user-agent=Mozilla/5.0 (iPhone; CPU iPhone OS 14_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.0.3 Mobile/15E148 Safari/604.1",
			"--window-size=375,812",
		},
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
	w.Wd.ResizeWindow("", 375, 812)
	if err = w.Wd.Get("https://home.m.jd.com/myJd/newhome.action"); err != nil {
		return err
	}

	return err
}

func (w *WdDriver) GetWd() selenium.WebDriver {
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
	return &WdDriver{
		Ct: ct,
	}
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
		return nil, errors.New("not init")
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
