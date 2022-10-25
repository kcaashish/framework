package http

import "github.com/sujit-baniya/framework/contracts/view"

type Json map[string]any

type Response interface {
	view.View
	String(code int, format string, values ...any) error
	Json(code int, obj any) error
	SendFile(filepath string, compress ...bool) error
	Download(filepath, filename string) error
	StatusCode() int
	SetHeader(key, value string) Context
	Vary(key string, value ...string)
}
