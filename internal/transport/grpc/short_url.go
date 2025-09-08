package grpc

import (
	"context"
	"time"

	"github.com/fsdevblog/shorturl/internal/transport/grpc/itrcpt"
	"github.com/fsdevblog/shorturl/internal/transport/trnptf"
	fdto "github.com/fsdevblog/shorturl/internal/transport/trnptf/dto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/fsdevblog/shorturl/internal/transport/grpc/proto"
)

const (
	defaultRequestTimeout = time.Second * 3
)

// ShortURL фасад для коротких ссылок.
type ShortURL struct {
	urlFacade *trnptf.URL
	pb.UnimplementedShortURLServer
}

// New создает новый экземпляр класса ShortURL.
func New(urlFacade *trnptf.URL) *ShortURL {
	return &ShortURL{
		urlFacade: urlFacade,
	}
}

// CreateShortURL Создание короткой ссылки.
func (s *ShortURL) CreateShortURL(
	ctx context.Context,
	request *pb.CreateShortURLRequest,
) (*pb.CreateShortURLResponse, error) {
	visitorUUID, _ := ctx.Value(itrcpt.VisitorUUIDKey).(string)

	reqCtx, cancel := context.WithTimeout(ctx, defaultRequestTimeout)
	defer cancel()

	shortURL, _, err := s.urlFacade.CreateShortURL(reqCtx, visitorUUID, request.GetUrl())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create short URL: %v", err)
	}
	return &pb.CreateShortURLResponse{Result: shortURL}, nil
}

// BatchCreate массовое создание коротких ссылок.
func (s *ShortURL) BatchCreate(ctx context.Context, req *pb.BatchCreateRequest) (*pb.BatchCreateResponse, error) {
	visitorUUID, _ := ctx.Value(itrcpt.VisitorUUIDKey).(string)
	reqCtx, cancel := context.WithTimeout(ctx, defaultRequestTimeout)
	defer cancel()

	var params = make([]fdto.BatchCreateParams, len(req.GetUrls()))
	for i, url := range req.GetUrls() {
		params[i] = fdto.BatchCreateParams{
			CorrelationID: url.GetCorrelationId(),
			OriginalURL:   url.GetOriginalUrl(),
		}
	}
	result, err := s.urlFacade.BatchCreate(reqCtx, visitorUUID, params)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create short URLs: %v", err)
	}
	var response = make([]*pb.BatchCreateResult, len(result))
	for i, r := range result {
		var eStr string
		if r.Error != nil {
			eStr = r.Error.Error()
		}
		response[i] = &pb.BatchCreateResult{
			CorrelationId: r.CorrelationID,
			ShortUrl:      r.ShortURL,
			Error:         eStr,
		}
	}
	return &pb.BatchCreateResponse{Result: response}, nil
}

// DeleteUserURLs удаляет ссылки текущего юзера.
func (s *ShortURL) DeleteUserURLs(ctx context.Context, req *pb.DeleteUserURLsRequest) (*pb.EmptyResponse, error) {
	visitorUUID, _ := ctx.Value(itrcpt.VisitorUUIDKey).(string)
	reqCtx, cancel := context.WithTimeout(ctx, defaultRequestTimeout)
	defer cancel()
	if err := s.urlFacade.DeleteUserURLs(reqCtx, visitorUUID, req.GetIds()); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete user URLs: %v", err)
	}
	return &pb.EmptyResponse{}, nil
}

// UserURLs выборка ссылок текущего юзера.
func (s *ShortURL) UserURLs(ctx context.Context, _ *pb.EmptyRequest) (*pb.UserURLsResponse, error) {
	visitorUUID, _ := ctx.Value(itrcpt.VisitorUUIDKey).(string)
	reqCtx, cancel := context.WithTimeout(ctx, defaultRequestTimeout)
	defer cancel()

	result, err := s.urlFacade.UserURLs(reqCtx, visitorUUID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user URLs: %v", err)
	}
	var response = make([]*pb.UserUrls, len(result))
	for i, r := range result {
		response[i] = &pb.UserUrls{
			ShortUrl:    r.ShortURL,
			OriginalUrl: r.OriginalURL,
		}
	}
	return &pb.UserURLsResponse{Urls: response}, nil
}
