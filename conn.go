package win

import (
	"errors"
	"github.com/gorilla/websocket"
	"log"
	"sync"
)

type Conn struct {
	Id         uint32
	Server     *Server
	Conn       *websocket.Conn
	msgHandler *msgHandler
	pool       sync.Pool
	sendChan   chan Response
	exitChan   chan bool
	mu         sync.Mutex
	closing    bool
	storeMu    sync.Mutex
	store      map[string]interface{}
}

func NewConn(server *Server, id uint32, conn *websocket.Conn, msgHandler *msgHandler) *Conn {
	c := &Conn{
		Id:         id,
		Server:     server,
		Conn:       conn,
		msgHandler: msgHandler,
		sendChan:   make(chan Response),
		exitChan:   make(chan bool),
	}
	c.pool.New = func() interface{} {
		return NewContext(nil, nil)
	}
	c.Server.register(id, c)
	return c
}

func (c *Conn) Start() {
	log.Printf("[win-debug]: Conn %d start", c.Id)
	go c.readMessages()
	go c.writeMessages()
	if c.Server.connStartCallback != nil {
		c.Server.connStartCallback(c)
	}
}

func (c *Conn) Close() {
	defer log.Printf("[win-debug]: Conn %d closed", c.Id)
	c.mu.Lock()
	if c.closing {
		c.mu.Unlock()
		return
	}
	c.closing = true
	c.mu.Unlock()

	c.exitChan <- true

	if c.Server.connCloseCallback != nil {
		c.Server.connCloseCallback(c)
	}

	c.Conn.Close()
	c.Server.connId = 0
	c.Server.remove(c.Id)
	close(c.exitChan)
	close(c.sendChan)
}

// 读goroutine
func (c *Conn) readMessages() {
	log.Printf("[win-debug]: goroutine readMessages runing")
	defer func() {
		c.Close()
		log.Printf("[win-debug]: goroutine readMessages close")
	}()
	for {
		var request Request
		err := c.Conn.ReadJSON(&request)
		if err != nil {
			if !websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[win-debug]: Conn has closed: %v", err)
				break
			} else {
				log.Printf("[win-debug]: read message error: %v", err)
				continue
			}
		}

		// 从对象池里取context
		ctx := c.pool.Get().(*Context)
		ctx.reset(&request, c)

		// 使用Worker池
		if c.msgHandler.workerPoolSize > 0 {
			c.msgHandler.SendToTaskQueue(*ctx)
		} else {
			go c.msgHandler.DoHandler(*ctx)
		}

		c.pool.Put(ctx)
	}
}

// 写goroutine
func (c *Conn) writeMessages() {
	log.Printf("[win-debug]: goroutine writeMessages runing")
	defer func() {
		log.Printf("[win-debug]: goroutine writeMessages close")
	}()
	for {
		select {
		case <-c.exitChan:
			return
		case resp, ok := <-c.sendChan:
			if !ok {
				log.Printf("[win-debug]: sendChan closed")
				return
			}
			c.Conn.WriteJSON(resp)
		}
	}
}

// 发送数据
func (c *Conn) SendMessage(resp Response) {
	c.mu.Lock()
	if c.closing {
		c.mu.Unlock()
		log.Printf("[win-debug]: Conn closed when send message")
		return
	}
	c.mu.Unlock()
	c.sendChan <- resp
}

// 取值
func (c *Conn) Get(key string) (interface{}, error) {
	defer c.storeMu.Unlock()
	if val, ok := c.store[key]; ok {
		return val, nil
	}
	return nil, errors.New("Not exist key: " + key)
}

// 设置值
func (c *Conn) Set(key string, val interface{}) {
	c.storeMu.Lock()
	defer c.storeMu.Unlock()
	if c.store == nil {
		c.store = make(map[string]interface{})
	}
	c.store[key] = val
}
