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
	msgHandler *MsgHandler
	pool       sync.Pool
	sendChan   chan Response
	exitChan   chan bool
	closing    bool
	mu         sync.Mutex
	sending    sync.Mutex
	store      map[string]interface{}
}

func NewConn(server *Server, id uint32, conn *websocket.Conn, msgHandler *MsgHandler) *Conn {
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
	log.Printf("[wnet-debug]: Conn %d start", c.Id)
	go c.readMessages()
	go c.writeMessages()
}

func (c *Conn) Close() {
	c.mu.Lock()
	defer log.Printf("[wnet-debug]: Conn %d closed", c.Id)
	if c.closing {
		c.mu.Unlock()
		return
	}
	c.closing = true
	c.mu.Unlock()

	c.exitChan <- true

	c.Conn.Close()
	c.Server.connId.Release(c.Id)
	c.Server.remove(c.Id)
	close(c.exitChan)
	close(c.sendChan)
}

// 读goroutine
func (c *Conn) readMessages() {
	log.Printf("[wnet-debug]: goroutine readMessages runing")
	defer func() {
		c.Close()
		log.Printf("[wnet-debug]: goroutine readMessages close")
	}()
	for {
		var request Request
		err := c.Conn.ReadJSON(&request)
		if err != nil {
			if !websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[wnet-debug]: Conn has closed: %v", err)
				break
			} else {
				log.Printf("[wnet-debug]: read message error: %v", err)
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
			c.msgHandler.DoHandler(*ctx)
		}

		c.pool.Put(ctx)
	}
}

// 写goroutine
func (c *Conn) writeMessages() {
	log.Printf("[wnet-debug]: goroutine writeMessages runing")
	defer func() {
		log.Printf("[wnet-debug]: goroutine writeMessages close")
	}()
	for {
		select {
		case <-c.exitChan:
			return
		case resp, ok := <-c.sendChan:
			if !ok {
				log.Printf("[wnet-debug]: sendChan closed")
				return
			}
			c.sending.Lock()
			c.Conn.WriteJSON(resp)
			c.sending.Unlock()
		}
	}
}

// 发送数据
func (c *Conn) SendMessage(resp Response) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closing {
		log.Printf("[wnet-debug]: Conn closed when send message")
		return
	}
	c.sendChan <- resp
}

// 取值
func (c *Conn) Get(key string) (interface{}, error) {
	defer c.mu.Unlock()
	if val, ok := c.store[key]; ok {
		return val, nil
	}
	return nil, errors.New("Not exist key: " + key)
}

// 设置值
func (c *Conn) Set(key string, val interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.store == nil {
		c.store = make(map[string]interface{})
	}
	c.store[key] = val
}
