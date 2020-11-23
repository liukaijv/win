package win

import (
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
)

type Server struct {
	mu                sync.RWMutex
	clients           map[uint32]*Conn
	msgHandler        *msgHandler
	upgrader          *websocket.Upgrader
	connId            uint32
	connStartCallback func(conn *Conn)
	connCloseCallback func(conn *Conn)
	handshakeHandler  func(r *http.Request) bool
	notFoundHandler   HandlerFunc
}

func NewServer() *Server {
	s := &Server{
		clients:    make(map[uint32]*Conn),
		msgHandler: newMsgHandler(),
		upgrader: newUpgrader(config.allowedOrigins, func(r *http.Request) bool {
			return true
		}),
		connId: 0,
	}

	s.msgHandler.notFoundHandler = s.notFoundHandler

	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	// 开启Workers
	if config.workerPoolSize > 0 {
		go s.msgHandler.startWorkerPool()
	}
	return s
}

func (s *Server) Serve(w http.ResponseWriter, r *http.Request) {
	c, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[win-debug]: websocket upgrader err: %v", err)
		return
	}

	if len(s.clients) > config.maxConn {
		log.Printf("[win-debug]: max connections limit: %d", config.maxConn)
		c.Close()
		return
	}

	s.connId++
	conn := newConn(s, s.connId, c, s.msgHandler)
	log.Printf("[win-debug]: new conn, id: %d", s.connId)

	go conn.start()
}

func (s *Server) Close() {
	log.Printf("[win-debug]: Server Close")
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, conn := range s.clients {
		conn.Close()
		delete(s.clients, id)
	}
}

func (s *Server) register(id uint32, conn *Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients[id] = conn
}

func (s *Server) remove(id uint32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.clients, id)
}

func (s *Server) Len() int {
	return len(s.clients)
}

func (s *Server) Use(middleware ...MiddlewareFunc) {
	s.msgHandler.use(middleware...)
}

func (s *Server) AddHandler(name string, h HandlerFunc, middleware ...MiddlewareFunc) {
	s.msgHandler.handlerFunc(name, h, middleware...)
}

func (s *Server) Group(prefix string, middleware ...MiddlewareFunc) Group {
	return newGroup(prefix, s, middleware...)
}

func (s *Server) SetConnStartHandler(handler func(conn *Conn)) {
	s.connStartCallback = handler
}

func (s *Server) SetConnCloseHandler(handler func(conn *Conn)) {
	s.connCloseCallback = handler
}

func (s *Server) SetHandshakeHandler(handler func(r *http.Request) bool) {
	s.handshakeHandler = handler
}

func (s *Server) SetNotFoundHandler(handler HandlerFunc) {
	s.notFoundHandler = handler
}

func isAllowedOrigin(r *http.Request, allowedOrigins []*regexp.Regexp) bool {
	origin := r.Header.Get("origin")
	if origin == "" {
		return true
	}

	u, err := url.Parse(origin)
	if err != nil {
		return false
	}

	if strings.ToLower(u.Host) == strings.ToLower(r.Host) {
		return true
	}

	for _, allowedOrigin := range allowedOrigins {
		if allowedOrigin.Match([]byte(strings.ToLower(u.Hostname()))) {
			return true
		}
	}

	return false
}

func newUpgrader(allowedWebSocketOrigins []string, handshakeHandler func(r *http.Request) bool) *websocket.Upgrader {
	compiledAllowedOrigins := compileAllowedWebSocketOrigins(allowedWebSocketOrigins)
	return &websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return isAllowedOrigin(r, compiledAllowedOrigins) && handshakeHandler(r)
		},
	}
}

func compileAllowedWebSocketOrigins(allowedOrigins []string) []*regexp.Regexp {
	var compiledAllowedOrigins []*regexp.Regexp
	for _, origin := range allowedOrigins {
		compiledAllowedOrigins = append(compiledAllowedOrigins, regexp.MustCompile(origin))
	}

	return compiledAllowedOrigins
}
