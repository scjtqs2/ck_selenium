package app

import (
	"embed"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/guonaihong/gout"
	"github.com/guonaihong/gout/dataflow"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/sessions"
	"github.com/kataras/iris/v12/sessions/sessiondb/boltdb"
	log "github.com/sirupsen/logrus"
	"github.com/stitch-june/selenium"
	"go.uber.org/dig"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

type SeSessionTime struct {
	T  int64
	Se SeInterface
}

type HttpServer struct {
	engine  *iris.Application
	HTTP    *http.Server
	ct      *dig.Container
	Se      map[string]*SeSessionTime
	Mux     sync.Mutex
	Session *sessions.Sessions
}

type MoveBody struct {
	Type string  `json:"type"` // 鼠标类型
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
	T    int64   `json:"t"` // 间隔时间
}

func NewHttp(ct *dig.Container) *HttpServer {
	var s HttpServer
	s.Mux = sync.Mutex{}
	s.Se = make(map[string]*SeSessionTime)
	ct.Invoke(func(static embed.FS) {
		f = static
	})

	db, err := boltdb.New("./tmp/sessions.db", os.FileMode(0750))
	if err != nil {
		panic(err)
	}

	// close and unlobkc the database when control+C/cmd+C pressed
	iris.RegisterOnInterrupt(func() {
		db.Close()
	})

	defer db.Close() // close and unlock the database if application errored.
	sess := sessions.New(sessions.Config{
		Cookie:       "sessionscookieid",
		Expires:      2 * time.Hour, // <=0 means unlimited life. Defaults to 0.
		AllowReclaim: true,
	})

	//
	// IMPORTANT:
	//
	sess.UseDatabase(db)
	s.Session = sess
	s.engine = iris.New()
	s.ct = ct
	s.engine.Use(sess.Handler())
	// s.engine.Use(func(c iris.Context) {
	// 	if c.Method() != "GET" && c.Method() != "POST" {
	// 		log.Warnf("已拒绝客户端 %v 的请求: 方法错误", c.Request().RemoteAddr)
	// 		c.NotFound()
	// 		return
	// 	}
	// 	c.Next()
	// })
	// 自动加载模板
	s.engine.RegisterView(iris.HTML("static/html", ".html").Binary(Asset, AssetNames))
	// s.engine.SetHTMLTemplate(templ)
	s.engine.Get("/", func(context iris.Context) {
		s.initSeBySession(context)
		context.View("passwordlogin.html")
	})
	s.engine.Get("/captcha", s.getCaptcha)
	s.engine.Post("/captchactions", s.actions)
	s.engine.Get("/screenshort", func(context iris.Context) {
		s.initSeBySession(context)
		context.View("screenshort.html")
	})
	s.engine.Get("/screenshortnow", s.getscreenShort)
	s.engine.Get("/pagesource", s.getPageSource)
	s.engine.Get("/checkcookie", s.checkCookie)       // 校验cookie
	s.engine.Post("/nomalLogin", s.nomalLogin)        // 账号密码方式登录
	s.engine.Get("/secondsms", s.secondsms)           // 二次短信认证
	s.engine.Get("/entersecondsms", s.entersecondsms) // 二次短信认证
	s.engine.Get("/exit", s.exit)
	go func() {
		port := "9999"
		if os.Getenv("HTTP_PORT") != "" {
			port = os.Getenv("HTTP_PORT")
		}
		err = s.engine.Run(iris.Addr(":" + port))
		if err != nil {
			log.Fatalf("error init http listen port %s err:%v", port, err)
		}
	}()
	go s.cleanSes()
	return &s
}

func (h *HttpServer) getCaptcha(ctx iris.Context) {
	se := h.getSeFromCtx(ctx)
	if se == nil {
		ctx.JSON(map[string]interface{}{
			"code": 0,
			"msg":  "not init",
		})
		return
	}
	ca, err := se.GetCaptcha()
	if err != nil {
		ctx.Problem(err)
		return
	}
	if ca == nil {
		ctx.Problem(errors.New("no captcha"))
		return
	}
	ctx.ViewData("ImgSrc", ca.Src)
	ctx.ViewData("x", ca.X)
	ctx.ViewData("y", ca.Y)
	ctx.ViewData("tips", ca.Tips)
	ctx.View("captcha.html")
}

func (h *HttpServer) actions(c iris.Context) {
	var reqInfo []MoveBody
	err := c.ReadJSON(&reqInfo)
	if err != nil {
		c.Problem(err)
		return
	}
	action := make([]selenium.PointerAction, 0)
	rand.Seed(time.Now().UnixMilli())
	for _, body := range reqInfo {
		x, _ := strconv.Atoi(fmt.Sprintf("%1.0f", body.X))
		y, _ := strconv.Atoi(fmt.Sprintf("%1.0f", body.Y))
		switch body.Type {
		case "start":
			action = append(action, selenium.PointerMoveAction(0, selenium.Point{X: x, Y: y}, selenium.FromViewport))
			action = append(action, selenium.PointerDownAction(selenium.LeftButton))
		case "end":
			action = append(action, selenium.PointerPauseAction(time.Microsecond*time.Duration(body.T)))
			action = append(action, selenium.PointerMoveAction(0, selenium.Point{X: x, Y: y}, selenium.FromViewport))
			action = append(action, selenium.PointerUpAction(selenium.LeftButton))
		case "move":
			action = append(action, selenium.PointerPauseAction(time.Microsecond*time.Duration(body.T)))
			action = append(action, selenium.PointerMoveAction(0, selenium.Point{X: x, Y: y}, selenium.FromViewport))
		}
	}
	se := h.getSeFromCtx(c)
	if se == nil {
		c.JSON(map[string]interface{}{
			"code": 0,
			"msg":  "not init",
		})
		return
	}
	h.getSeFromCtx(c).CheckCaptcha2(action)
	c.JSON(map[string]interface{}{
		"code": 0,
	})
}

func (h *HttpServer) getscreenShort(c iris.Context) {
	se := h.getSeFromCtx(c)
	if se == nil {
		c.StatusCode(http.StatusForbidden)
		return
	}
	imgByte, err := h.getSeFromCtx(c).GetScreenShort()
	if err != nil {
		c.Problem(err)
		return
	}
	src := fmt.Sprintf("data:image/png;base64,%s", base64.StdEncoding.EncodeToString(imgByte))
	c.JSON(map[string]interface{}{
		"src": src,
	})
}

func (h *HttpServer) getPageSource(c iris.Context) {
	se := h.getSeFromCtx(c)
	if se == nil {
		c.JSON(map[string]interface{}{
			"code": 0,
			"msg":  "not init",
		})
		return
	}
	src, err := h.getSeFromCtx(c).GetSource()
	if err != nil {
		c.Problem(err)
		return
	}
	c.HTML(src)
}

// 推送到远程服务器
func (h *HttpServer) PostWebHookCk(cookie string) (string, error) {
	var webhook WebHook
	var err error
	var msg string
	// //发送数据给 挂机服务器
	h.ct.Invoke(func(hook WebHook) {
		webhook = hook
	})
	postUrl := webhook.Url
	if postUrl != "" {
		var res string
		code := 0
		var flow *dataflow.DataFlow
		switch webhook.Method {
		case "GET":
			flow = gout.GET(webhook.Url).SetQuery(gout.H{
				webhook.Key: cookie,
			})
			break
		case "POST":
			flow = gout.POST(postUrl).SetWWWForm(
				gout.H{
					webhook.Key: cookie,
				},
			)
			break
		default:
			flow = gout.POST(postUrl)
			break
		}
		err = flow.
			Debug(false).
			BindBody(&res).
			SetHeader(gout.H{
				"Connection":   "Keep-Alive",
				"Content-Type": "application/x-www-form-urlencoded; Charset=UTF-8",
				"Accept":       "application/json, text/plain, */*",
				"User-Agent":   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/86.0.4240.111 Safari/537.36",
			}).
			Code(&code).
			SetTimeout(timeout).
			F().Retry().Attempt(5).
			WaitTime(time.Millisecond * 500).MaxWaitTime(time.Second * 5).
			Do()
		if err != nil || code != 200 {
			log.Errorf("upsave notify post  usercookie to %s, res=%s faild err=%v, http_code=%d", postUrl, res, err, code)
			msg = fmt.Sprintf("upsave notify post  usercookie to %s, res=%s faild err=%v, http_code=%d", postUrl, res, err, code)
		} else {
			log.Infof("upsave to url %s post usercookie=%s success res=%s", postUrl, cookie, res)
			msg = fmt.Sprintf("upsave to url %s post usercookie=%s success res=%s", postUrl, cookie, res)
		}
		return msg, err
	}
	return msg, err
}

func (h *HttpServer) checkCookie(ctx iris.Context) {
	se := h.getSeFromCtx(ctx)
	if se == nil {
		ctx.StatusCode(http.StatusForbidden)
		ctx.JSON(map[string]interface{}{
			"code": 0,
			"msg":  "not init",
		})
		return
	}
	ck, err := h.getSeFromCtx(ctx).GetCookie()
	if err != nil {
		log.Errorf("get cookie error,%v", err)
		ctx.JSON(map[string]interface{}{
			"code": 400,
			"msg":  err.Error(),
		})
		return
	}
	if ck != "" {
		log.Infof("cookie %s", ck)
		msg, err := h.PostWebHookCk(ck)
		if err != nil {
			ctx.JSON(map[string]interface{}{
				"code": 400,
				"msg":  msg,
			})
			return
		}
		defer h.getSeFromCtx(ctx).Quit()
		if msg != "" && err == nil {
			ctx.JSON(map[string]interface{}{
				"code": 0,
				"msg":  msg,
			})
		}
	}
}

func (h *HttpServer) nomalLogin(ctx iris.Context) {
	type Login struct {
		Name   string `json:"name"`
		Passwd string `json:"passwd"`
	}
	var reqInfo Login

	err := ctx.ReadJSON(&reqInfo)
	if err != nil {
		ctx.JSON(map[string]interface{}{
			"code": 400,
			"msg":  err.Error(),
		})
		return
	}
	h.initSeBySession(ctx)
	se := h.getSeFromCtx(ctx)
	if se == nil || se.GetWd() == nil {
		ctx.JSON(map[string]interface{}{
			"code": 0,
			"msg":  "not init",
		})
		return
	}
	err = h.getSeFromCtx(ctx).SeRun()
	if err != nil {
		ctx.JSON(map[string]interface{}{
			"code": 400,
			"msg":  err.Error(),
		})
		return
	}
	time.Sleep(2 * time.Second)
	err = h.getSeFromCtx(ctx).ChangeLoginType()
	if err != nil {
		ctx.JSON(map[string]interface{}{
			"code": 400,
			"msg":  err.Error(),
		})
		return
	}
	err = h.getSeFromCtx(ctx).EnterUserName(reqInfo.Name)
	if err != nil {
		ctx.JSON(map[string]interface{}{
			"code": 400,
			"msg":  err.Error(),
		})
		return
	}
	err = h.getSeFromCtx(ctx).EnterPasswd(reqInfo.Passwd)
	if err != nil {
		ctx.JSON(map[string]interface{}{
			"code": 400,
			"msg":  err.Error(),
		})
		return
	}
	err = h.getSeFromCtx(ctx).SubmitLogin()
	if err != nil {
		ctx.JSON(map[string]interface{}{
			"code": 400,
			"msg":  err.Error(),
		})
		return
	}

	ctx.JSON(map[string]interface{}{
		"code": 0,
		"msg":  "ok",
	})
}

func (h *HttpServer) secondsms(ctx iris.Context) {
	se := h.getSeFromCtx(ctx)
	if se == nil {
		ctx.JSON(map[string]interface{}{
			"code": 0,
			"msg":  "not init",
		})
		return
	}
	h.getSeFromCtx(ctx).SecondSmsCheck()
	h.getSeFromCtx(ctx).SecondSmsSend()
	ctx.View("secondsms.html")
}

func (h *HttpServer) entersecondsms(ctx iris.Context) {
	smscode := ctx.Params().Get("smscode")
	se := h.getSeFromCtx(ctx)
	if se == nil {
		ctx.JSON(map[string]interface{}{
			"code": 0,
			"msg":  "not init",
		})
		return
	}
	err := h.getSeFromCtx(ctx).EnterSecondSmsCode(smscode)
	if err != nil {
		ctx.JSON(map[string]interface{}{
			"code": 400,
			"msg":  err.Error(),
		})
		return
	} else {
		ctx.JSON(map[string]interface{}{
			"code": 0,
			"msg":  "ok",
		})
	}
}

func (h *HttpServer) exit(ctx iris.Context) {
	se := h.getSeFromCtx(ctx)
	if se == nil {
		ctx.JSON(map[string]interface{}{
			"code": 0,
			"msg":  "not init",
		})
		return
	}
	defer h.getSeFromCtx(ctx).Quit()
	ck, err := h.getSeFromCtx(ctx).GetCookie()
	if err != nil {
		log.Errorf("get cookie error,%v", err)
		ctx.JSON(map[string]interface{}{
			"code": 400,
			"msg":  err.Error(),
		})
		return
	}
	if ck != "" {
		log.Infof("cookie %s", ck)
		msg, err := h.PostWebHookCk(ck)
		if err != nil {
			ctx.JSON(map[string]interface{}{
				"code": 400,
				"msg":  msg,
			})
			return
		}
		if msg != "" && err == nil {
			ctx.JSON(map[string]interface{}{
				"code": 0,
				"msg":  msg,
			})
		}
	}
}

func (h *HttpServer) initSeBySession(ctx iris.Context) {
	session := sessions.Get(ctx)
	id := session.ID()
	if h.getSe(id) == nil || h.getSe(id).GetWd() != nil {
		h.setSe(id, NewWdService(h.ct))
	}
	return
}

func (h *HttpServer) getSe(id string) SeInterface {
	defer h.Mux.Unlock()
	h.Mux.Lock()
	if se, ok := h.Se[id]; ok {
		return se.Se
	}
	return nil
}

func (h *HttpServer) getAllSe() map[string]*SeSessionTime {
	defer h.Mux.Unlock()
	h.Mux.Lock()
	all := make(map[string]*SeSessionTime)
	for s, sessionTime := range h.Se {
		all[s] = sessionTime
	}
	return all
}

func (h *HttpServer) setSe(id string, se SeInterface) {
	defer h.Mux.Unlock()
	h.Mux.Lock()
	h.Se[id] = &SeSessionTime{
		T:  time.Now().Unix(),
		Se: se,
	}
}

func (h *HttpServer) getSeFromCtx(ctx iris.Context) SeInterface {
	session := sessions.Get(ctx)
	id := session.ID()
	return h.getSe(id)
}

func (h *HttpServer) cleanSes() {
	for {
		time.Sleep(time.Second * 5)
		all := h.getAllSe()
		for id, sessionTime := range all {
			t := time.Now().Unix()
			if (t - sessionTime.T) > 60*20 {
				sessionTime.Se.Quit()
				h.setSe(id, sessionTime.Se)
			}
		}
	}
}
