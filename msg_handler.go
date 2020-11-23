package win

import (
	"log"
)

type msgHandler struct {
	handlers        map[string]HandlerFunc
	middleware      []MiddlewareFunc
	workerPoolSize  uint32
	taskQueue       []chan Context
	notFoundHandler HandlerFunc
}

func newMsgHandler() *msgHandler {
	m := &msgHandler{
		handlers:       make(map[string]HandlerFunc),
		middleware:     make([]MiddlewareFunc, 0),
		workerPoolSize: config.workerPoolSize,
		taskQueue:      make([]chan Context, config.workerPoolSize),
	}
	return m
}

// 使用中间件
func (m *msgHandler) use(middleware ...MiddlewareFunc) {
	m.middleware = append(m.middleware, middleware...)
}

func (m *msgHandler) handlerFunc(name string, h HandlerFunc, middleware ...MiddlewareFunc) {
	if _, ok := m.handlers[name]; ok {
		panic("Repeated handler name: " + name)
	}
	log.Printf("[win-debug]: add handler %s", name)
	m.handlers[name] = func(ctx Context) {
		h = applyMiddleware(h, middleware...)
		h(ctx)
	}
}

func (m *msgHandler) doHandler(c Context) {
	log.Printf("[win-debug]: invoke handler %s", c.Request.Method)
	h, ok := m.handlers[c.Request.Method]
	if !ok {
		log.Printf("[win-debug]: invoke handler name %s not exists", c.Request.Method)
		if m.notFoundHandler != nil {
			m.notFoundHandler(c)
		} else {
			c.ReplyError(404, "not found")
		}
		return
	}
	h = applyMiddleware(h, m.middleware...)
	h(c)
}

func (m *msgHandler) sendToTaskQueue(c Context) {
	workerId := c.Conn.id % m.workerPoolSize
	log.Printf("[win-debug]: send message to worker, worker id: %d", workerId)
	m.taskQueue[workerId] <- c
}

func (m *msgHandler) startWorker(i uint32, taskQueue chan Context) {
	for {
		select {
		case ctx := <-taskQueue:
			m.doHandler(ctx)
		}
	}
}

func (m *msgHandler) startWorkerPool() {
	log.Printf("[win-debug]: start worker poll, size: %d", m.workerPoolSize)
	var i uint32
	for i = 0; i < m.workerPoolSize; i++ {
		m.taskQueue[i] = make(chan Context, config.workerTaskMax)
		go m.startWorker(i, m.taskQueue[i])
	}
}

func applyMiddleware(h HandlerFunc, middleware ...MiddlewareFunc) HandlerFunc {
	for i := len(middleware) - 1; i >= 0; i-- {
		h = middleware[i](h)
	}
	return h
}
