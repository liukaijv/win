package win

import (
	"fmt"
	"strings"
)

type Group struct {
	prefix     string
	server     *Server
	middleware []MiddlewareFunc
	parent     *Group
}

func newGroup(prefix string, server *Server, middleware ...MiddlewareFunc) Group {
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}
	return Group{
		prefix:     prefix,
		server:     server,
		middleware: middleware,
	}
}

func (g *Group) Use(middleware ...MiddlewareFunc) {
	g.middleware = append(g.middleware, middleware...)
}

func (g *Group) AddHandler(name string, h HandlerFunc, middleware ...MiddlewareFunc) {
	path := g.getPrefix() + "/" + name
	fmt.Println(path)
	path = strings.ReplaceAll(path, "//", "/")
	m := make([]MiddlewareFunc, len(g.middleware)+len(middleware))
	m = append(m, g.middleware...)
	m = append(m, middleware...)
	g.server.AddHandler(path, h, m...)
}

func (g *Group) Group(prefix string, middleware ...MiddlewareFunc) Group {
	if prefix == "/" {
		prefix = ""
	}
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}
	group := Group{
		prefix: prefix,
		server: g.server,
		parent: g,
	}
	group.middleware = make([]MiddlewareFunc, len(g.middleware)+len(middleware))
	group.middleware = append(group.middleware, g.middleware...)
	group.middleware = append(group.middleware, middleware...)
	return group
}

func (g *Group) getPrefix() string {
	prefix := g.prefix
	parent := g.parent
	for parent != nil {
		if parent.prefix != "/" {
			prefix = parent.prefix + prefix
		}
		parent = parent.parent
	}
	return prefix
}
