package win

import (
	"encoding/json"
)

type Context struct {
	Request *Request
	Conn    *Conn
}

func NewContext(r *Request, conn *Conn) *Context {
	return &Context{
		Request: r,
		Conn:    conn,
	}
}

func (c *Context) reset(r *Request, conn *Conn) {
	c.Request = r
	c.Conn = conn
}

func (c *Context) sendMessage(resp Response) {
	c.Conn.SendMessage(resp)
}

// 返回数据
func (c *Context) Reply(data interface{}) {
	resp := Response{
		Method: c.Request.Method,
		ID:     c.Request.ID,
		Error:  nil,
	}
	resp.setResult(data)
	c.sendMessage(resp)
}

// 返回错误信息
func (c *Context) ReplyError(code int, msg string, data ...interface{}) {
	var errData interface{}
	if len(data) > 0 {
		errData = data[0]
	}
	resp := Response{
		Method: c.Request.Method,
		ID:     c.Request.ID,
		Error: &Error{
			Code:    code,
			Message: msg,
			Data:    errData,
		},
	}
	c.sendMessage(resp)
}

// 推送给客户端的数据
func (c *Context) Notify(data interface{}) {
	resp := Response{
		Method: c.Request.Method,
		ID:     0,
		Error:  nil,
	}
	resp.setResult(data)
	c.sendMessage(resp)
}

func (c *Context) NotifyError(code int, msg string, data ...interface{}) {
	var errData interface{}
	if len(data) > 0 {
		errData = data[0]
	}
	resp := Response{
		Method: c.Request.Method,
		ID:     c.Request.ID,
		Error: &Error{
			Code:    code,
			Message: msg,
			Data:    errData,
		},
	}
	c.sendMessage(resp)
}

// 取值
func (c *Context) Get(key string) (interface{}, error) {
	return c.Conn.get(key)
}

// 设置值
func (c *Context) Set(key string, val interface{}) {
	c.Conn.set(key, val)
}

func (c *Context) BindJson(ptr interface{}) error {
	if err := json.Unmarshal(*c.Request.Params, ptr); err != nil {
		return err
	}
	return nil
}
