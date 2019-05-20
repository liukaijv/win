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
	mu         sync.RWMutex
	clients    map[uint32]*Conn
	msgHandler *MsgHandler
	upgrader   *websocket.Upgrader
	connId     *connId
}

func NewServer() *Server {
	s := &Server{
		clients:    make(map[uint32]*Conn),
		msgHandler: NewMsgHandler(),
		upgrader:   newUpgrader(config.AllowedOrigins),
		connId:     NewConnId(),
	}

	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	// 开启Workers
	if config.WorkerPoolSize > 0 {
		go s.msgHandler.StartWorkerPool()
	}
	return s
}

func (s *Server) Serve(w http.ResponseWriter, r *http.Request) {
	c, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[wnet-debug]: websocket upgrader err: %v", err)
		return
	}

	if len(s.clients) > config.MaxConn {
		log.Printf("[wnet-debug]: max connections limit: %d", config.MaxConn)
		c.Close()
		return
	}

	cid := s.connId.Get()
	conn := NewConn(s, cid, c, s.msgHandler)
	log.Printf("[wnet-debug]: new Conn, id: %d", cid)

	go conn.Start()
}

func (s *Server) Close() {
	log.Printf("[wnet-debug]: Server close")
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
	s.msgHandler.Use(middleware...)
}

func (s *Server) AddHandler(name string, h HandlerFunc, middleware ...MiddlewareFunc) {
	s.msgHandler.HandlerFunc(name, h, middleware...)
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

func newUpgrader(allowedWebSocketOrigins []string) *websocket.Upgrader {
	compiledAllowedOrigins := compileAllowedWebSocketOrigins(allowedWebSocketOrigins)
	return &websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return isAllowedOrigin(r, compiledAllowedOrigins)
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