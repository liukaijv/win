package win

import (
	"log"
)

type MsgHandler struct {
	handlers       map[string]HandlerFunc
	middleware     []MiddlewareFunc
	workerPoolSize uint32
	taskQueue      []chan Context
}

func NewMsgHandler() *MsgHandler {
	m := &MsgHandler{
		handlers:       make(map[string]HandlerFunc),
		middleware:     make([]MiddlewareFunc, 0),
		workerPoolSize: config.WorkerPoolSize,
		taskQueue:      make([]chan Context, config.WorkerPoolSize),
	}
	return m
}

// 使用中间件
func (m *MsgHandler) Use(middleware ...MiddlewareFunc) {
	m.middleware = append(m.middleware, middleware...)
}

func (m *MsgHandler) HandlerFunc(name string, h HandlerFunc, middleware ...MiddlewareFunc) {
	if _, ok := m.handlers[name]; ok {
		panic("Repeated handler name: " + name)
	}
	log.Printf("[wnet-debug]: add handler %s", name)
	m.handlers[name] = applyMiddleware(h, middleware...)
}

func (m *MsgHandler) DoHandler(c Context) {
	log.Printf("[wnet-debug]: invoke handler %s", c.Request.Method)
	h, ok := m.handlers[c.Request.Method]
	if !ok {
		log.Printf("[wnet-debug]: invoke handler name %s not exists", c.Request.Method)
		return
	}
	h = applyMiddleware(h, m.middleware...)
	h(c)
}

func (m *MsgHandler) SendToTaskQueue(c Context) {
	workerId := c.Conn.Id % m.workerPoolSize
	log.Printf("[wnet-debug]: send message to worker, worker id: %d", workerId)
	m.taskQueue[workerId] <- c
}

func (m *MsgHandler) StartWorker(i uint32, taskQueue chan Context) {
	for {
		select {
		case ctx := <-taskQueue:
			m.DoHandler(ctx)
		}
	}
}

func (m *MsgHandler) StartWorkerPool() {
	log.Printf("[wnet-debug]: start worker poll, size: %d", m.workerPoolSize)
	var i uint32
	for i = 0; i < m.workerPoolSize; i++ {
		m.taskQueue[i] = make(chan Context, config.WorkerTaskMax)
		go m.StartWorker(i, m.taskQueue[i])
	}
}

func applyMiddleware(h HandlerFunc, middleware ...MiddlewareFunc) HandlerFunc {

	for i := len(middleware) - 1; i >= 0; i-- {
		h = middleware[i](h)
	}

	return h
}
