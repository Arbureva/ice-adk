package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

// Set is an ordered, concurrency-safe collection of tools keyed by name. It is
// both the unit a caller advertises to a request (Definitions / RequestTools
// render into the native tool list) and the unit it dispatches model tool-calls
// against (Invoke). Registration order is preserved by Definitions, Tools, and
// Names.
type Set struct {
	mu    sync.RWMutex
	order []string
	tools map[string]Tool
}

// NewSet returns a Set seeded with the given tools, in order. It panics on a nil
// tool, an empty name, or a duplicate — tool sets are wired at startup, so a
// clash is a programming error, mirroring chat.Register.
func NewSet(tools ...Tool) *Set {
	s := &Set{tools: make(map[string]Tool, len(tools))}
	for _, t := range tools {
		s.Add(t)
	}
	return s
}

// Add registers t. It panics on a nil tool, an empty name, or a duplicate name.
func (s *Set) Add(t Tool) {
	if t == nil {
		panic("tool: Add nil tool")
	}
	name := t.Definition().Name
	if name == "" {
		panic("tool: Add tool with empty name")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, dup := s.tools[name]; dup {
		panic("tool: Add called twice for tool " + name)
	}
	s.tools[name] = t
	s.order = append(s.order, name)
}

// Has reports whether a tool is registered under name.
func (s *Set) Has(name string) bool {
	s.mu.RLock()
	_, ok := s.tools[name]
	s.mu.RUnlock()
	return ok
}

// Get returns the tool registered under name.
func (s *Set) Get(name string) (Tool, bool) {
	s.mu.RLock()
	t, ok := s.tools[name]
	s.mu.RUnlock()
	return t, ok
}

// Invoke dispatches a model tool-call: it looks up name and runs it with args.
// An unknown name yields a non-nil error (the host could not honour the call);
// a known tool's own failure surfaces through its Result/error per Tool.Invoke.
func (s *Set) Invoke(ctx context.Context, name string, args json.RawMessage) (*Result, error) {
	t, ok := s.Get(name)
	if !ok {
		return nil, fmt.Errorf("tool: no tool named %q", name)
	}
	return t.Invoke(ctx, args)
}

// Definitions returns the advertised definitions in registration order.
func (s *Set) Definitions() []Definition {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Definition, 0, len(s.order))
	for _, name := range s.order {
		out = append(out, s.tools[name].Definition())
	}
	return out
}

// Tools returns the registered tools in registration order.
func (s *Set) Tools() []Tool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Tool, 0, len(s.order))
	for _, name := range s.order {
		out = append(out, s.tools[name])
	}
	return out
}

// RequestTools boxes the registered tools for adapter.Request.Tools, which is
// loosely typed ([]interface{}) so the adapter package need not import tool:
//
//	req := adapter.Request{Provider: adapter.OpenAI, Data: nr, Tools: set.RequestTools()}
func (s *Set) RequestTools() []interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]interface{}, 0, len(s.order))
	for _, name := range s.order {
		out = append(out, s.tools[name])
	}
	return out
}

// Names returns the registered tool names in order.
func (s *Set) Names() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]string(nil), s.order...)
}

// Len reports how many tools are registered.
func (s *Set) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.order)
}
