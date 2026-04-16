package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Code 业务错误码
type Code int

const (
	CodeOK            Code = 0
	CodeInvalidParam  Code = 1001
	CodeUnauthorized  Code = 1002
	CodeForbidden     Code = 1003
	CodeNotFound      Code = 1004
	CodeUserExists    Code = 1005
	CodeWrongPassword Code = 1006
	CodeUserBanned    Code = 1007
	CodeInternalError Code = 5000
)

// R 统一响应体
type R struct {
	Code Code        `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

// OK 返回成功
func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, R{Code: CodeOK, Msg: "ok", Data: data})
}

// Fail 返回业务错误
func Fail(c *gin.Context, code Code, msg string) {
	c.JSON(http.StatusOK, R{Code: code, Msg: msg})
}

// FailWithStatus 返回指定 HTTP 状态码的错误（如 401、403）
func FailWithStatus(c *gin.Context, httpStatus int, code Code, msg string) {
	c.JSON(httpStatus, R{Code: code, Msg: msg})
}
