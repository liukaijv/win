package win

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"sync"
	"time"
)

type (
	ClientHandler func(response Response)
)

type CallOpt struct {
	Timeout uint
	Headers map[string]interface{}
}

type call struct {
	request  *Request
	response *Response
	seq      int64 // the seq of the Request
	done     chan error
}

type Client struct {
	conn     *websocket.Conn
	exitChan chan bool
	seq      int64
	pending  map[int64]*call
	timeout  uint
	sending  sync.Mutex
	handlers map[string]ClientHandler
	mu       sync.Mutex
}

func Dial(urlStr string, requestHeader http.Header) (*Client, error) {
	c, _, err := websocket.DefaultDialer.Dial(urlStr, requestHeader)
	if err != nil {
		log.Printf("[win-debug]: websocket dial err: %v", err)
		return nil, err
	}

	cli := &Client{
		conn:     c,
		exitChan: make(chan bool),
		pending:  make(map[int64]*call),
		timeout:  5000,
	}

	go cli.readMessages()
	return cli, nil
}

func (c *Client) Close() {
	err := c.conn.Close()
	if err != nil {
		log.Printf("[win-debug]: client close err: %v", err)
	}
	c.exitChan <- true
	close(c.exitChan)

	log.Printf("[win-debug]: client closed")
}

func (c *Client) AddHandler(name string, h ClientHandler) {
	if c.handlers == nil {
		c.handlers = make(map[string]ClientHandler)
	}
	if _, ok := c.handlers[name]; ok {
		panic("Repeated handler name: " + name)
	}
	c.handlers[name] = h
	log.Printf("[win-debug]: add handler %s", name)
}

func (c *Client) readMessages() {
	log.Printf("[win-debug]: goroutine readMessages runing")
	defer log.Printf("[win-debug]: goroutine readMessages closed")
	for {
		var resp Response
		err := c.conn.ReadJSON(&resp)
		if err != nil {
			if !websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[win-debug]: Conn has closed: %v", err)
				break
			} else {
				log.Printf("[win-debug]: read message error: %v", err)
				continue
			}
		}
		c.handleResponse(resp)
		select {
		case <-c.exitChan:
			break
		default:
		}
	}

	c.mu.Lock()
	for _, call := range c.pending {
		call.done <- errors.New("client close")
		close(call.done)
	}
	c.mu.Unlock()
}

func (c *Client) handleResponse(resp Response) {
	if resp.ID == 0 {
		c.mu.Lock()
		if h, ok := c.handlers[resp.Method]; ok {
			h(resp)
		} else {
			log.Printf("[win-debug] ignoring response %s with no handler", resp.Method)
		}
		c.mu.Unlock()
	} else {
		id := resp.ID
		c.mu.Lock()
		call := c.pending[id]
		delete(c.pending, id)
		c.mu.Unlock()

		if call != nil {
			call.response = &resp
		}

		switch {
		case call == nil:
			log.Printf("[win-debug] ignoring response %d with no corresponding request", id)
		case resp.Error != nil:
			call.done <- resp.Error
			close(call.done)
		default:
			call.done <- nil
			close(call.done)
		}
	}

}

func (c *Client) sendMessage(request *Request, wait bool) (cc *call, err error) {
	c.sending.Lock()
	defer c.sending.Unlock()

	var id int64

	if wait {
		c.seq++
		cc = &call{request: request, seq: c.seq, done: make(chan error, 1)}
		if request.ID == 0 {
			request.ID = c.seq
		}
		id = request.ID
		c.pending[id] = cc
	}

	defer func() {
		if err != nil {
			if cc != nil {
				c.mu.Lock()
				delete(c.pending, id)
				c.mu.Unlock()
			}
		}
	}()

	err = c.conn.WriteJSON(request)
	if err != nil {
		return nil, err
	}
	return cc, nil
}

// 发起请求，阻塞到数据返回或超时
func (c *Client) Call(method string, params, reply interface{}, opts ... *CallOpt) error {
	req := Request{
		Method: method,
	}
	err := req.SetParams(params)
	if err != nil {
		return err
	}

	var opt *CallOpt
	if len(opts) > 0 {
		opt = opts[0]
	}

	if opt != nil && len(opt.Headers) > 0 {
		req.SetHeaders(opt.Headers)
	}

	call, err := c.sendMessage(&req, true)
	if err != nil {
		return err
	}

	timeout := c.timeout
	if opt != nil && opt.Timeout > 0 {
		timeout = opt.Timeout
	}

	t := time.NewTimer(time.Millisecond * time.Duration(timeout))
	select {
	case err, ok := <-call.done:
		if !ok {
			return errors.New("Conn has closed")
		}
		if err != nil {
			return err
		}
		if call.response.Result == nil {
			call.response.Result = &jsonNull
		}
		if err := json.Unmarshal(*call.response.Result, reply); err != nil {
			return err
		}
		return nil
	case <-t.C:
		return errors.New(fmt.Sprintf("Request name [%s] timeout %d ms.\n", method, c.timeout))
	}
	return nil
}

// 发送不需要返回
func (c *Client) Notify(method string, params interface{}, opts ... *CallOpt) error {
	req := Request{
		Method: method,
	}
	err := req.SetParams(params)
	if err != nil {
		return err
	}
	var opt *CallOpt
	if len(opts) > 0 {
		opt = opts[0]
	}

	if opt != nil && len(opt.Headers) > 0 {
		req.SetHeaders(opt.Headers)
	}
	_, err = c.sendMessage(&req, false)
	return err
}
