package testutils

import (
	"context"
	"fmt"
	"net"

	pb "github.com/fsdevblog/shorturl/internal/transport/grpc/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

// Client реализация тестового GRPC клиента.
type Client struct {
	conn   *grpc.ClientConn
	Client pb.ShortURLClient
}

// NewClient создает нового GRPC клиента.
func NewClient(ctx context.Context, lis *bufconn.Listener) (*Client, error) {
	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Client: %w", err)
	}

	client := pb.NewShortURLClient(conn)

	return &Client{
		conn:   conn,
		Client: client,
	}, nil
}

// Close закрывает текущее соединение.
func (c *Client) Close() error {
	if err := c.conn.Close(); err != nil {
		return fmt.Errorf("failed to close grpc client: %w", err)
	}
	return nil
}
