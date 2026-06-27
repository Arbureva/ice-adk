package tool

import (
	"context"
	"encoding/json"
)

// Definition is the provider-agnostic description of a tool exposed to a model:
// a name, an LLM-facing description, and a JSON Schema for the arguments object.
// chat drivers render it into each provider's native tool shape (OpenAI/DeepSeek
// function tools, Anthropic input_schema tools), so a tool is declared once and
// works across providers.
type Definition struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Schema      json.RawMessage `json:"schema,omitempty"`
}

// Tool is the single abstraction the whole SDK agrees on. A tool is a callable:
// it advertises a Definition (what the model sees) and an Invoke that actually
// runs it. Higher-level packages — mcp, skills, cli, ... — implement Tool to put
// their own execution model behind this one interface; nothing downstream of a
// Tool needs to know which kind it is holding.
type Tool interface {
	// Definition returns the name / description / schema advertised to the model.
	Definition() Definition

	// Invoke runs the tool with the model-supplied arguments (raw JSON, exactly
	// as the model produced them) and returns a Result.
	//
	// The two failure channels are distinct. A non-nil error means the call
	// could not be carried out — bad host wiring, transport down, panic — and is
	// the host's problem. A tool that ran but failed on its own terms (bad
	// input, upstream 404, ...) returns a *Result with IsError set and a nil
	// error, so the model sees the failure and can react instead of the loop
	// aborting.
	Invoke(ctx context.Context, args json.RawMessage) (*Result, error)
}

// Handler is the function bound inside a Func tool. It receives the raw argument
// JSON and returns a Result. This is deliberately the lowest common denominator:
// richer shapes (typed parameters, struct returns, client-side delegation, MCP
// proxying) are wrappers a caller layers on top, never a separate tool kind in
// this package.
type Handler func(ctx context.Context, args json.RawMessage) (*Result, error)

// funcTool is the built-in Tool that binds a Definition to a Handler.
type funcTool struct {
	def Definition
	fn  Handler
}

// Func builds a Tool from a name, description, argument schema, and handler.
// schema is the JSON Schema of the arguments object; pass nil for a no-argument
// tool. fn must not be nil (Func panics otherwise, since a tool with no body is
// a wiring bug).
//
// Write the schema by hand, or derive it from a Go type with Reflect:
//
//	type Args struct {
//	    City  string `json:"city"            desc:"City name, e.g. Paris"`
//	    Units string `json:"units,omitempty" desc:"c or f"`
//	}
//	t := tool.Func("get_weather", "Look up current weather", tool.Reflect(Args{}),
//	    func(ctx context.Context, raw json.RawMessage) (*tool.Result, error) {
//	        var a Args
//	        if err := json.Unmarshal(raw, &a); err != nil {
//	            return tool.Errf("bad arguments: %v", err), nil
//	        }
//	        return tool.Textf("It is sunny in %s.", a.City), nil
//	    })
func Func(name, description string, schema json.RawMessage, fn Handler) Tool {
	if fn == nil {
		panic("tool: Func handler is nil")
	}
	return &funcTool{
		def: Definition{Name: name, Description: description, Schema: schema},
		fn:  fn,
	}
}

func (t *funcTool) Definition() Definition { return t.def }

func (t *funcTool) Invoke(ctx context.Context, args json.RawMessage) (*Result, error) {
	return t.fn(ctx, args)
}
