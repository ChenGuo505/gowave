package gowave

import "strings"

type Trie struct {
	Root *Node
}

func NewTrie() *Trie {
	return &Trie{
		Root: &Node{text: "/", children: make([]*Node, 0)},
	}
}

func (t *Trie) Put(text string) {
	if t.Root == nil {
		t.Root = &Node{text: "/", children: make([]*Node, 0)}
	}
	t.Root.Put(text)
}

func (t *Trie) Get(text string) *Node {
	if t.Root == nil {
		return nil
	}
	return t.Root.Get(text)
}

type Node struct {
	text       string
	routerName string
	children   []*Node
	isEnd      bool
}

func (n *Node) Put(text string) {
	cur := n
	strList := strings.Split(text, "/")
	routerName := ""
	for idx, str := range strList {
		if idx == 0 {
			continue
		}
		children := cur.children
		isMatch := false
		for _, child := range children {
			if child.text == str {
				routerName += "/" + child.text
				isMatch = true
				cur = child
				break
			}
		}
		if !isMatch {
			isEnd := false
			if idx == len(strList)-1 {
				isEnd = true
			}
			routerName += "/" + str
			node := &Node{text: str, routerName: routerName, children: make([]*Node, 0), isEnd: isEnd}
			children = append(children, node)
			cur.children = children
			cur = node
		}
	}
}

func (n *Node) Get(text string) *Node {
	cur := n
	strList := strings.Split(text, "/")
	routerName := ""
	for idx, str := range strList {
		if idx == 0 {
			continue
		}
		children := cur.children
		isMatch := false
		for _, child := range children {
			if child.text == str || child.text == "*" || strings.Contains(child.text, ":") {
				isMatch = true
				routerName += "/" + child.text
				child.routerName = routerName
				cur = child
				if idx == len(strList)-1 {
					return child
				}
				break
			}
		}
		if !isMatch {
			for _, child := range children {
				if child.text == "**" {
					routerName += "/" + child.text
					child.routerName = routerName
					return child
				}
			}
		}
	}
	return nil
}
