package enum

import "net/http"

// RetMsg ...
var RetMsg = map[int]string{
	RetNormal:     "正常",
	RetAbnormal:   "异常",
	RetProxyError: "代理相关的异常",
}

// CodeMsg ...
var CodeMsg = map[int]string{
	CodeSuccessed:      "Success",
	ECodeRouteNotFound: "未能成功匹配代理路由",
	ECodeProxyFailed:   "源服务器错误",
}

// CodeStatus ...
var CodeStatus = map[int]int{
	CodeSuccessed:      http.StatusOK,
	ECodeRouteNotFound: http.StatusNotFound,
	ECodeNotFile:       http.StatusForbidden,
	ECodeDirNotEmpty:   http.StatusForbidden,
	ECodeUnauthorized:  http.StatusUnauthorized,
	ECodeTestFailed:    http.StatusPreconditionFailed,
	ECodeProxyFailed:   http.StatusBadGateway,
	ECodeInternal:      http.StatusInternalServerError,
}

const (
	// RetNormal 正常返回
	RetNormal = 0
	// RetAbnormal 不正常返回
	RetAbnormal = 1

	// RetProxyError 代理相关错误
	RetProxyError = 5
)

const (
	// CodeSuccessed 成功
	CodeSuccessed = 0
	// CodeRouteNotFound 未匹配代理路由
	ECodeRouteNotFound = 100
	ECodeNotFile       = 101
	ECodeDirNotEmpty   = 102
	ECodeUnauthorized  = 103
	ECodeTestFailed    = 104
	ECodeProxyFailed   = 105
	ECodeInternal      = 106
)
