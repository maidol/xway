package authtoken

import (
	"database/sql"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
	"xway/context"
	"xway/enum"
	xerror "xway/error"
	"xway/plugin"
	"xway/plugin/handler"
	xcrypto "xway/utils/crypto"

	"github.com/garyburd/redigo/redis"
	"github.com/mholt/binding"
	"github.com/urfave/negroni"
)

type AuthToken struct {
	// 统一处理错误
	handler.StdErrorHandler
	opt Options
}

type Options struct {
}

type QueryData struct {
	ClientId string
	Token    string
}

// 可添加自定义验证
// 可集成https://github.com/asaskevich/govalidator
// func (qd *QueryData) Validate(req *http.Request) error {
// 	if qd.Token == "" {
// 		return binding.Errors{
// 			binding.NewError([]string{"accessToken"}, "EmptyError", "accessToken 不能为空"),
// 		}
// 	}
// 	return nil
// }

func (qd *QueryData) FieldMap(req *http.Request) binding.FieldMap {
	return binding.FieldMap{
		&qd.ClientId: binding.Field{
			Form:         "clientId",
			Required:     true,
			ErrorMessage: " clitenId不能为空",
		},
		&qd.Token: binding.Field{
			Form:         "accessToken",
			Required:     true,
			ErrorMessage: " accessToken不能为空",
		},
	}
}

type HeaderData struct {
	ClientId string
	TimeLine int64
	Sign     string
}

func (hd *HeaderData) FieldMap(req *http.Request) binding.FieldMap {
	return binding.FieldMap{
		&hd.ClientId: binding.Field{
			Form:         "Clientid",
			Required:     true,
			ErrorMessage: " clientid不能为空",
		},
		&hd.TimeLine: binding.Field{
			Form:         "Timeline",
			Required:     true,
			ErrorMessage: " timeline不能为空, 且必须为数值",
		},
		&hd.Sign: binding.Field{
			Form:         "Sign",
			Required:     true,
			ErrorMessage: " sign不能为空",
		},
	}
}

type appClient struct {
	clientId   string
	privateKey string
	status     int
}

// New ...
// 创建中间件实例
func New(opt interface{}) negroni.Handler {
	o, ok := opt.(Options)
	if !ok {
		o = Options{}
	}
	return &AuthToken{
		opt: o,
	}
}

// 验证clientId的存在和有效性
func clientAuth(clientId string, registry *plugin.Registry) (*appClient, error) {
	db := registry.GetDBPool()
	ac := &appClient{}
	row := db.QueryRow("select clientId, privateKey, status from apps where clientId=?", clientId)
	if err := row.Scan(&ac.clientId, &ac.privateKey, &ac.status); err != nil {
		if err == sql.ErrNoRows {
			e := xerror.NewRequestError(enum.RetAbnormal, enum.ECodeClientException, err.Error())
			return nil, e
		}
		e := xerror.NewRequestError(enum.RetAbnormal, enum.ECodeInternal, "row.Scan err "+err.Error())
		return nil, e
	}
	if ac.status != 0 {
		e := xerror.NewRequestError(enum.RetAbnormal, enum.ECodeClientException, "client.status!=0")
		return ac, e
	}
	return ac, nil
}

func getToken(token string, registry *plugin.Registry) (map[string]string, error) {
	p := registry.GetRedisPool()
	// fmt.Println(p.ActiveCount(), p.IdleCount(), p.Stats())
	rdc := p.Get()
	defer func() {
		// 重要: 释放客户端
		if err := rdc.Close(); err != nil {
			// TODO: 处理错误, 记录日志
			fmt.Printf("[AuthToken getToken] rdc.Close err: %v\n", err)
		}
	}()
	// 读取token, 验证权限
	tk := "cw:gateway:token:" + token
	v, err := rdc.Do("HGETALL", tk)
	if err != nil {
		e := xerror.NewRequestError(enum.RetAbnormal, enum.ECodeInternal, err.Error())
		return nil, e
	}
	m, err := redis.StringMap(v, err)
	if err != nil {
		e := xerror.NewRequestError(enum.RetAbnormal, enum.ECodeInternal, err.Error())
		return nil, e
	}
	return m, nil
}

func (at *AuthToken) accessToken(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	xwayCtx := xwaycontext.DefaultXWayContext(r.Context())
	// 验证请求参数
	qd := new(QueryData)
	if errs := binding.URL(r, qd); errs != nil {
		e := xerror.NewRequestError(enum.RetAbnormal, enum.ECodeParamsError, "请求url信息 "+errs.Error())
		at.RequestError(rw, r, e)
		return
	}
	// TODO: 需要验证clientId的存在和有效性
	if _, err := clientAuth(qd.ClientId, xwayCtx.Registry); err != nil {
		at.RequestError(rw, r, err)
		return
	}

	// token
	m, err := getToken(qd.Token, xwayCtx.Registry)
	if err != nil {
		at.RequestError(rw, r, err)
		return
	}

	if m == nil || len(m) == 0 {
		e := xerror.NewRequestError(enum.RetAbnormal, enum.ECodeUnauthorized, "未找到有效token")
		at.RequestError(rw, r, e)
		return
	}
	if m["clientId"] != qd.ClientId {
		e := xerror.NewRequestError(enum.RetAbnormal, enum.ECodeParamsError, "clientId与accessToken不对应")
		at.RequestError(rw, r, e)
		return
	}
	expireDate, err := strconv.Atoi(m["expireDate"])
	if err != nil {
		e := xerror.NewRequestError(enum.RetAbnormal, enum.ECodeInternal, err.Error()+` [strconv.Atoi(m["expireDate"])转换失败]`)
		at.RequestError(rw, r, e)
		return
	}
	if int64(expireDate) < time.Now().Unix() {
		e := xerror.NewRequestError(enum.RetAbnormal, enum.ECodeUnauthorized, "token已过期")
		at.RequestError(rw, r, e)
		return
	}
	r.SetBasicAuth(m["userId"], "123456")
	next(rw, r)
}

func (at *AuthToken) clientCredentials(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	// 验证请求参数
	hd := new(HeaderData)
	if errs := binding.Header(r, hd); errs != nil {
		e := xerror.NewRequestError(enum.RetAbnormal, enum.ECodeParamsError, "请求头部信息 "+errs.Error())
		at.RequestError(rw, r, e)
		return
	}
	// 比较时间戳
	if math.Abs(float64(time.Now().Unix()-int64(hd.TimeLine))) > 180 { //时间戳允许最大误差值为±180秒
		e := xerror.NewRequestError(enum.RetAbnormal, enum.ECodeParamsError, "参数timeLine验证失败,请检查服务器时间")
		at.RequestError(rw, r, e)
		return
	}
	// TODO: (计划优化)比较签名clientId, timeLine, sign, path, query
	// 查询clientInfo(目前from mysql)
	xwayCtx := xwaycontext.DefaultXWayContext(r.Context())
	// TODO: 需要验证clientId的存在和有效性
	app, err := clientAuth(hd.ClientId, xwayCtx.Registry)
	if err != nil {
		at.RequestError(rw, r, err)
		return
	}

	text := generateOriginalText4Sign(hd, r)
	if s, b := checkHamcSign(text, hd.Sign, app.privateKey); !b {
		e := xerror.NewRequestError(enum.RetAbnormal, enum.ECodeHmacsha1SignError, "sign签名不匹配: "+hd.Sign+", 正确签名: "+s+", 原始值: "+text)
		at.RequestError(rw, r, e)
		return
	}

	next(rw, r)
}

func checkHamcSign(content, sign string, key string) (string, bool) {
	s := xcrypto.HmacSha1(content, key, "hex")
	if s == sign {
		return s, true
	}
	return s, false
}

func generateOriginalText4Sign(hd *HeaderData, r *http.Request) string {
	timeLineKey := "timeLine"
	pathKey := "path"
	pathVal := r.URL.Path
	if strings.Index(pathVal, "/gateway/") == 0 {
		pathVal = strings.Replace(pathVal, "/gateway/", "/", 1)
	}
	originalObj := map[string]string{timeLineKey: strconv.FormatInt(hd.TimeLine, 10), pathKey: pathVal}
	keys := []string{timeLineKey, pathKey}
	vals := []string{}
	qs := r.URL.Query()
	for k, strs := range qs {
		v := strs[0]
		originalObj[k] = v
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, v := range keys {
		vals = append(vals, v+"="+originalObj[v])
	}
	text := strings.Join(vals, "&")
	return text
}

func (at *AuthToken) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	grantType := r.URL.Query().Get("grantType")
	switch grantType {
	case "":
		fallthrough
	case "accesstoken":
		at.accessToken(rw, r, next)
	case "clientcredentials":
		at.clientCredentials(rw, r, next)
	default:
		e := xerror.NewRequestError(enum.RetAbnormal, enum.ECodeUnauthorized, "不支持的授权模式: "+grantType)
		at.RequestError(rw, r, e)
		return
	}
}
