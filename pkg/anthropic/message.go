package anthropic

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
)

// Chat performs a non-streaming Messages request and returns the assistant
// message. Stream is forced false.
func (c *Client) Chat(ctx context.Context, req *Request) (*Message, error) {
	if req == nil {
		return nil, fmt.Errorf("anthropic: nil request")
	}
	req.Stream = false

	resp, err := c.do(ctx, req, false)
	if err != nil {
		return nil, err
	}
	defer drainAndClose(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("anthropic: read response: %w", err)
	}

	var msg Message
	if err := json.Unmarshal(body, &msg); err != nil {
		return nil, fmt.Errorf("anthropic: decode response: %w", err)
	}
	return &msg, nil
}
