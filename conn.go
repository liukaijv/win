package win

import (
	"errors"
	"github.com/gorilla/websocket"
	"log"
	"sync"
)

type Conn struct {
	id         uint32
	Server     *Server
	conn       *websocket.Conn
	msgHandler *msgHandler
	pool       sync.Pool
	sendChan   chan Response
	exitChan   chan bool
	mu         sync.Mutex
	closing    bool
	storeMu    sync.Mutex
	store      map[string]interface{}
}

func newConn(server *Server, id uint32, conn *websocket.Conn, msgHandler *msgHandler) *Conn {
	c := &Conn{
		id:         id,
		Server:     server,
		conn:       conn,
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

func (c *Conn) start() {
	log.Printf("[win-debug]: conn %d start", c.id)
	go c.readMessages()
	go c.writeMessages()
	if c.Server.connStartCallback != nil {
		c.Server.connStartCallback(c)
	}
}

func (c *Conn) Close() {
	defer log.Printf("[win-debug]: conn %d closed", c.id)
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

	c.conn.Close()
	c.Server.connId = 0
	c.Server.remove(c.id)
	close(c.exitChan)
	close(c.sendChan)
}

// 读goroutine
func (c *Conn) readMessages() {
	log.Printf("[win-debug]: goroutine readMessages runing")
	defer func() {
		c.Close()
		log.Printf("[win-debug]: goroutine readMessages Close")
	}()
	for {
		var request Request
		err := c.conn.ReadJSON(&request)
		if err != nil {
			if !websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[win-debug]: conn has closed: %v", err)
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
			c.msgHandler.sendToTaskQueue(*ctx)
		} else {
			go c.msgHandler.doHandler(*ctx)
		}

		c.pool.Put(ctx)
	}
}

// 写goroutine
func (c *Conn) writeMessages() {
	log.Printf("[win-debug]: goroutine writeMessages runing")
	defer func() {
		log.Printf("[win-debug]: goroutine writeMessages Close")
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
			c.conn.WriteJSON(resp)
		}
	}
}

// 发送数据
func (c *Conn) SendMessage(resp Response) {
	c.mu.Lock()
	if c.closing {
		c.mu.Unlock()
		log.Printf("[win-debug]: conn closed when send message")
		return
	}
	c.mu.Unlock()
	c.sendChan <- resp
}

// 取值
func (c *Conn) get(key string) (interface{}, error) {
	defer c.storeMu.Unlock()
	if val, ok := c.store[key]; ok {
		return val, nil
	}
	return nil, errors.New("Not exist key: " + key)
}

// 设置值
func (c *Conn) set(key string, val interface{}) {
	c.storeMu.Lock()
	defer c.storeMu.Unlock()
	if c.store == nil {
		c.store = make(map[string]interface{})
	}
	c.store[key] = val
}
