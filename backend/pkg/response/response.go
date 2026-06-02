// Package response 提供统一 API 响应格式与业务错误码
package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// 业务错误码
const (
	CodeSuccess      = 0
	CodeInvalidParam = 40001
	CodeUnauthorized = 40101
	CodeForbidden    = 40301
	CodeStockShort   = 40901
	CodeRateLimited  = 42901
	CodeServerError  = 50001
)

type Resp struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// Success 返回成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Resp{Code: CodeSuccess, Message: "ok", Data: data})
}

// Error 返回业务错误响应
func Error(c *gin.Context, httpStatus, code int, message string) {
	c.JSON(httpStatus, Resp{Code: code, Message: message, Data: nil})
}

// InvalidParam 参数无效
func InvalidParam(c *gin.Context, msg string) {
	Error(c, http.StatusBadRequest, CodeInvalidParam, msg)
}

// Unauthorized 未登录
func Unauthorized(c *gin.Context) {
	Error(c, http.StatusUnauthorized, CodeUnauthorized, "请先登录")
}

// Forbidden 权限不足
func Forbidden(c *gin.Context) {
	Error(c, http.StatusForbidden, CodeForbidden, "权限不足")
}

// TooManyRequests 请求过频
func TooManyRequests(c *gin.Context) {
	Error(c, http.StatusTooManyRequests, CodeRateLimited, "请求过于频繁,请稍后再试")
}

// ServerError 服务异常
func ServerError(c *gin.Context, msg string) {
	Error(c, http.StatusInternalServerError, CodeServerError, msg)
}
