package win

import "strings"

type Group struct {
	prefix     string
	server     *Server
	middleware []MiddlewareFunc
}

func newGroup(prefix string, server *Server) Group {
	return Group{
		prefix:     prefix,
		server:     server,
		middleware: []MiddlewareFunc{},
	}
}

func (g *Group) Use(middleware ...MiddlewareFunc) {
	g.middleware = middleware
}

func (g *Group) AddHandler(name string, h HandlerFunc, middleware ...MiddlewareFunc) {
	path := g.prefix + "/" + name
	strings.ReplaceAll(path, "//", "/")
	middleWares := g.middleware[:]
	middleWares = append(middleWares, middleware...)
	g.server.msgHandler.HandlerFunc(path, h, middleWares...)
}
