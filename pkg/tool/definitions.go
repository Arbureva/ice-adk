package tool

// Definer is implemented by anything that can describe itself as a tool, most
// notably Tool. chat drivers depend on this (not on Tool directly) so that a
// request can carry live tools through the loosely-typed adapter.Request.Tools
// slice without the adapter package importing tool.
type Definer interface {
	Definition() Definition
}

// DefinitionsOf normalizes the loosely-typed adapter.Request.Tools slice into
// concrete Definitions. Each element may be a Tool / Definer, a Definition, or a
// *Definition. The second return is false if any element is none of these, so a
// driver can surface a type mismatch instead of silently dropping a tool.
//
// An empty or nil input is reported as ok with a nil result.
func DefinitionsOf(tools []interface{}) ([]Definition, bool) {
	if len(tools) == 0 {
		return nil, true
	}
	out := make([]Definition, 0, len(tools))
	for _, raw := range tools {
		switch v := raw.(type) {
		case Definer:
			out = append(out, v.Definition())
		case Definition:
			out = append(out, v)
		case *Definition:
			if v != nil {
				out = append(out, *v)
			}
		default:
			return nil, false
		}
	}
	return out, true
}
