// Package steam reads Steam's local files (libraries, app manifests, cloud
// cache) to discover installed games and their save folders.
package steam

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// Node is one KeyValues pair: either a string value or a block of children.
type Node struct {
	Key      string
	Value    string
	Children []*Node
	Block    bool
}

// Child returns the first child whose key matches (case-insensitive).
func (n *Node) Child(key string) *Node {
	if n == nil {
		return nil
	}
	for _, c := range n.Children {
		if strings.EqualFold(c.Key, key) {
			return c
		}
	}
	return nil
}

// Get returns the value of child key, or "" when missing.
func (n *Node) Get(key string) string {
	if c := n.Child(key); c != nil {
		return c.Value
	}
	return ""
}

// ReadVDF parses the KeyValues file at path.
func ReadVDF(path string) (*Node, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	n, err := ParseVDF(f)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return n, nil
}

// ParseVDF parses Valve KeyValues text into a synthetic root block.
func ParseVDF(r io.Reader) (*Node, error) {
	p := &vdfParser{r: bufio.NewReader(r), line: 1}
	root := &Node{Block: true}
	if err := p.readBlock(root, false); err != nil {
		return nil, err
	}
	return root, nil
}

type tokenKind int

const (
	tokEOF tokenKind = iota
	tokString
	tokOpen
	tokClose
)

type vdfParser struct {
	r    *bufio.Reader
	line int
}

func (p *vdfParser) errorf(format string, args ...any) error {
	return fmt.Errorf("vdf line %d: %s", p.line, fmt.Sprintf(format, args...))
}

// readBlock reads key/value pairs into parent until "}" (nested) or EOF (top).
func (p *vdfParser) readBlock(parent *Node, nested bool) error {
	for {
		kind, key, err := p.next()
		if err != nil {
			return err
		}
		switch kind {
		case tokEOF:
			if nested {
				return p.errorf("missing closing brace")
			}
			return nil
		case tokClose:
			if !nested {
				return p.errorf("unexpected closing brace")
			}
			return nil
		case tokOpen:
			return p.errorf("block without key")
		}
		kind, value, err := p.next()
		if err != nil {
			return err
		}
		child := &Node{Key: key}
		switch kind {
		case tokString:
			child.Value = value
		case tokOpen:
			child.Block = true
			if err := p.readBlock(child, true); err != nil {
				return err
			}
		default:
			return p.errorf("key %q has no value", key)
		}
		parent.Children = append(parent.Children, child)
	}
}

func (p *vdfParser) next() (tokenKind, string, error) {
	for {
		c, err := p.r.ReadByte()
		if err == io.EOF {
			return tokEOF, "", nil
		}
		if err != nil {
			return tokEOF, "", err
		}
		switch {
		case c == '\n':
			p.line++
		case c == ' ' || c == '\t' || c == '\r':
		case c == '{':
			return tokOpen, "", nil
		case c == '}':
			return tokClose, "", nil
		case c == '"':
			s, err := p.quoted()
			return tokString, s, err
		case c == '/':
			if next, _ := p.r.Peek(1); len(next) == 1 && next[0] == '/' {
				p.r.ReadString('\n')
				p.line++
				continue
			}
			return tokEOF, "", p.errorf("unexpected '/'")
		case isBare(c):
			return tokString, p.bare(c), nil
		default:
			return tokEOF, "", p.errorf("unexpected character %q", c)
		}
	}
}

func (p *vdfParser) quoted() (string, error) {
	var b strings.Builder
	for {
		c, err := p.r.ReadByte()
		if err != nil {
			return "", p.errorf("unterminated string")
		}
		switch c {
		case '"':
			return b.String(), nil
		case '\n':
			p.line++
		case '\\':
			esc, err := p.r.ReadByte()
			if err != nil {
				return "", p.errorf("unterminated string")
			}
			switch esc {
			case 'n':
				c = '\n'
			case 't':
				c = '\t'
			default:
				c = esc
			}
		}
		b.WriteByte(c)
	}
}

func isBare(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '.' || c == '-'
}

func (p *vdfParser) bare(first byte) string {
	b := []byte{first}
	for {
		next, err := p.r.Peek(1)
		if err != nil || !isBare(next[0]) {
			return string(b)
		}
		p.r.ReadByte()
		b = append(b, next[0])
	}
}
