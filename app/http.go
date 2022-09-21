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
	"embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/tebeka/selenium"
	"go.uber.org/dig"
	"html/template"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"
)

type HttpServer struct {
	engine *gin.Engine
	HTTP   *http.Server
	ct     *dig.Container
	Se     SeInterface
}

type MoveBody struct {
	Type string  `json:"type"`
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
}

func NewHttp(ct *dig.Container) *HttpServer {
	var s HttpServer
	var f embed.FS
	ct.Invoke(func(static embed.FS) {
		f = static
	})
	gin.SetMode(gin.ReleaseMode)
	s.engine = gin.New()

	s.engine.Use(func(c *gin.Context) {
		if c.Request.Method != "GET" && c.Request.Method != "POST" {
			log.Warnf("已拒绝客户端 %v 的请求: 方法错误", c.Request.RemoteAddr)
			c.Status(404)
			return
		}
		c.Next()
	})
	// 自动加载模板
	// t := template.New("tmp")
	// 从二进制中加载模板（后缀必须.html)
	templ := template.Must(template.New("").ParseFS(f, "static/html/*.html"))
	s.engine.SetHTMLTemplate(templ)
	s.engine.GET("/captcha", s.getCaptcha)
	s.engine.POST("/captchactions", s.actions)
	s.engine.GET("/screenshort", func(context *gin.Context) {
		context.HTML(http.StatusOK, "screenshort.html", nil)
	})
	s.engine.GET("/screenshortnow", s.getscreenShort)
	s.engine.GET("/pagesource", s.getPageSource)
	go func() {
		s.HTTP = &http.Server{
			Addr:    fmt.Sprintf(":%d", 9999),
			Handler: s.engine,
		}
		if err := s.HTTP.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error(err)
			log.Infof("HTTP 服务启动失败, 请检查端口是否被占用.")
			log.Warnf("将在五秒后退出.")
			time.Sleep(time.Second * 5)
			os.Exit(1)
		}
	}()
	return &s
}

func (h *HttpServer) getCaptcha(ctx *gin.Context) {
	ca, err := h.Se.GetCaptcha()
	if err != nil {
		ctx.Error(err)
		ctx.Abort()
	}
	ctx.HTML(http.StatusOK, "captcha.html", gin.H{
		"ImgSrc": ca.Src,
		"x":      ca.X,
		"y":      ca.Y,
		"tips":   ca.Tips,
	})
}

func (h *HttpServer) actions(c *gin.Context) {
	var reqInfo []MoveBody
	b, err := io.ReadAll(c.Request.Body)
	println(string(b))
	defer c.Request.Body.Close()
	if err != nil {
		c.Error(err)
		c.Abort()
	}
	json.Unmarshal(b, &reqInfo)
	action := make([]selenium.PointerAction, 0)
	rand.Seed(time.Now().UnixMilli())
	for _, body := range reqInfo {
		x, _ := strconv.Atoi(fmt.Sprintf("%1.0f", body.X))
		y, _ := strconv.Atoi(fmt.Sprintf("%1.0f", body.Y))
		switch body.Type {
		case "start":
			action = append(action, selenium.PointerMoveAction(6, selenium.Point{X: x, Y: y}, selenium.FromViewport))
			action = append(action, selenium.PointerPauseAction(time.Duration(rand.Intn(250))))
			action = append(action, selenium.PointerDownAction(selenium.LeftButton))
			action = append(action, selenium.PointerPauseAction(time.Duration(rand.Intn(250))))
		case "end":
			action = append(action, selenium.PointerMoveAction(6, selenium.Point{X: x, Y: y}, selenium.FromViewport))
			action = append(action, selenium.PointerPauseAction(time.Duration(rand.Intn(250))))
			action = append(action, selenium.PointerUpAction(selenium.LeftButton))
		case "move":
			action = append(action, selenium.PointerMoveAction(6, selenium.Point{X: x, Y: y}, selenium.FromPointer))
			action = append(action, selenium.PointerPauseAction(time.Duration(rand.Intn(250))))
		}
	}
	h.Se.CheckCaptcha2(action)
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
	})
}

func (h *HttpServer) getscreenShort(c *gin.Context) {
	imgByte, err := h.Se.GetScreenShort()
	if err != nil {
		c.Error(err)
		return
	}
	src := fmt.Sprintf("data:image/png;base64,%s", base64.StdEncoding.EncodeToString(imgByte))
	c.JSON(http.StatusOK, gin.H{
		"src": src,
	})
}

func (h *HttpServer) getPageSource(c *gin.Context) {
	src, err := h.Se.GetSource()
	if err != nil {
		c.Error(err)
		return
	}
	c.Data(http.StatusOK, "html", []byte(src))
}
