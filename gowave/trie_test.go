package gowave

import (
	"testing"
)

func TestTrie(t *testing.T) {
	root := &Node{text: "/", children: make([]*Node, 0)}
	root.Put("/api/user/:id")
	root.Put("/api/info/hello")
	root.Put("/api/info/test")
	root.Put("/api/order/*")

	if root.Get("/api/user/123").text != ":id" {
		t.Errorf("Expected :id, got %s", root.Get("/api/user/123").text)
	}
	if root.Get("/api/info/hello").text != "hello" {
		t.Errorf("Expected hello, got %s", root.Get("/api/info/hello").text)
	}
	if root.Get("/api/info/test").text != "test" {
		t.Errorf("Expected test, got %s", root.Get("/api/info/test").text)
	}
	if root.Get("/api/order/123").text != "*" {
		t.Errorf("Expected *, got %s", root.Get("/api/order/123").text)
	}
}
