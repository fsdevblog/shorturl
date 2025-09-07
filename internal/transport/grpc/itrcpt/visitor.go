package itrcpt

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// ContextKey ключ контекста.
type ContextKey string

// VisitorUUIDKey ключ для uuid посетителя.
const VisitorUUIDKey ContextKey = "visitor_uuid"

// Visitor интерцептор. Кладет в контекст uuid посетителя.
// UUID берет из метадаты, если там его нет, - генерирует рандомный.
func Visitor(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	var visitorUUID string
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		values := md.Get(string(VisitorUUIDKey))
		if len(values) > 0 {
			visitorUUID = values[0]
		}
	}
	if visitorUUID == "" {
		vuuid, err := uuid.NewRandom()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to generate visitor UUID: %v", err)
		}
		visitorUUID = vuuid.String()
	}

	ctx = context.WithValue(ctx, VisitorUUIDKey, visitorUUID)
	return handler(ctx, req)
}
