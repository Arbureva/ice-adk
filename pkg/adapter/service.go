package adapter

import "context"

type ChatService interface {
	ChatStream(ctx context.Context, req Request) (<-chan ChunkMessageAdapter, error)
	Chat(ctx context.Context, req Request) (*MessageAdapter, error)
}
