package service

import "fmt"

// DomainError 携带传输无关的业务错误码与建议 HTTP 状态码，
// 供 HTTP handler 映射状态码、Wails service 直接返回给前端。
type DomainError struct {
	Status  int
	Code    string
	Message string
}

func (e *DomainError) Error() string {
	return e.Message
}

// Errorf 构造领域错误。
func Errorf(status int, code, format string, args ...any) *DomainError {
	return &DomainError{Status: status, Code: code, Message: fmt.Sprintf(format, args...)}
}
