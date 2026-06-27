package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
)

// Chat performs a non-streaming chat completion. Stream is forced false.
func (c *Client) Chat(ctx context.Context, req *Request) (*ChatCompletion, error) {
	if req == nil {
		return nil, fmt.Errorf("openai: nil request")
	}
	req.Stream = false
	req.StreamOptions = nil

	resp, err := c.do(ctx, req, false)
	if err != nil {
		return nil, err
	}
	defer drainAndClose(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("openai: read response: %w", err)
	}

	var out ChatCompletion
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("openai: decode response: %w", err)
	}
	return &out, nil
}
