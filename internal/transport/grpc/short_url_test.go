package grpc

import (
	"context"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/fsdevblog/shorturl/internal/models"
	"github.com/fsdevblog/shorturl/internal/services"
	"github.com/fsdevblog/shorturl/internal/transport/grpc/itrcpt"
	pb "github.com/fsdevblog/shorturl/internal/transport/grpc/proto"
	"github.com/fsdevblog/shorturl/internal/transport/grpc/testutils"
	"github.com/fsdevblog/shorturl/internal/transport/trnptf"
	"github.com/fsdevblog/shorturl/internal/transport/trnptf/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
)

type ShortURLSuite struct {
	suite.Suite
	mockCtrl        *gomock.Controller
	mockURLProvider *mocks.MockURLProvider
	server          *grpc.Server
	handler         *ShortURL

	grpcClient *testutils.Client
}

func TestShortURL(t *testing.T) {
	suite.Run(t, new(ShortURLSuite))
}

func (s *ShortURLSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockURLProvider = mocks.NewMockURLProvider(s.mockCtrl)

	urlFacade := trnptf.New(false, "localhost", "http://localhost", s.mockURLProvider)
	s.handler = New(urlFacade)

	lis := s.serverUp()
	time.Sleep(100 * time.Millisecond)
	client, err := testutils.NewClient(s.T().Context(), lis)

	if err != nil {
		s.T().Fatal(err)
	}
	s.grpcClient = client
}

func (s *ShortURLSuite) TearDownTest() {
	s.mockCtrl.Finish()

	if s.server != nil {
		s.server.Stop()
	}
	if s.grpcClient != nil {
		err := s.grpcClient.Close()
		s.Require().NoError(err)
	}
}

func (s *ShortURLSuite) TestCreateShortURL() {
	tests := []struct {
		name    string
		req     *pb.CreateShortURLRequest
		wantErr bool
	}{
		{
			name: "success",
			req: &pb.CreateShortURLRequest{
				Url:         gofakeit.URL(),
				VisitorUUID: gofakeit.UUID(),
			},
		}, {
			name: "invalid url",
			req: &pb.CreateShortURLRequest{
				Url:         "invalid url",
				VisitorUUID: gofakeit.UUID(),
			},
			wantErr: true,
		},
	}

	// Настраиваем мок сервиса
	s.mockURLProvider.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, visitorUUID string, url string) (*models.URL, bool, error) {
			shortURL := models.URL{
				ID:              1,
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
				DeletedAt:       nil,
				URL:             url,
				ShortIdentifier: gofakeit.Word(),
				VisitorUUID:     visitorUUID,
			}

			return &shortURL, true, nil
		})

	for _, tt := range tests {
		s.Run(tt.name, func() {
			md := metadata.New(map[string]string{string(itrcpt.VisitorUUIDKey): tt.req.GetVisitorUUID()})
			mdCtx := metadata.NewOutgoingContext(s.T().Context(), md)

			resp, err := s.grpcClient.Client.CreateShortURL(mdCtx, tt.req)

			if tt.wantErr {
				s.Require().Error(err)
				return
			}
			s.Require().NoError(err)
			s.NotEmpty(resp.GetResult())
		})
	}
}

func (s *ShortURLSuite) TestBatchCreate() {
	tests := []struct {
		name    string
		req     *pb.BatchCreateRequest
		wantErr bool
	}{
		{
			name: "success",
			req: &pb.BatchCreateRequest{
				Urls: []*pb.BatchCreateParams{
					{
						OriginalUrl:   gofakeit.URL(),
						CorrelationId: gofakeit.UUID(),
					},
					{
						OriginalUrl:   gofakeit.URL(),
						CorrelationId: gofakeit.UUID(),
					},
				},
			},
		}, {
			name: "with invalid url",
			req: &pb.BatchCreateRequest{
				Urls: []*pb.BatchCreateParams{
					{
						OriginalUrl:   "invalid url",
						CorrelationId: gofakeit.UUID(),
					},
				},
			},
			wantErr: true,
		},
	}

	// Подготавливаем мок
	s.mockURLProvider.EXPECT().
		BatchCreate(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(
			_ context.Context,
			visitorUUID string,
			urls []string,
		) (*services.BatchCreateShortURLsResponse, error) {
			resp := services.NewBatchExecResponse[models.URL](len(urls))
			var responseItems = make([]services.BatchResponseItem[models.URL], len(urls))
			for i, url := range urls {
				responseItems[i].Item = models.URL{
					ID:              gofakeit.Uint(),
					CreatedAt:       time.Now(),
					UpdatedAt:       time.Now(),
					DeletedAt:       nil,
					URL:             url,
					ShortIdentifier: gofakeit.Word(),
					VisitorUUID:     visitorUUID,
				}
			}
			resp.SetResults(responseItems)
			return services.NewBatchExecResponseURL(resp), nil
		})

	for _, tt := range tests {
		s.Run(tt.name, func() {
			md := metadata.New(map[string]string{string(itrcpt.VisitorUUIDKey): gofakeit.UUID()})
			mdCtx := metadata.NewOutgoingContext(s.T().Context(), md)
			resp, err := s.grpcClient.Client.BatchCreate(mdCtx, tt.req)
			if tt.wantErr {
				s.Require().Error(err)
				return
			}
			s.Require().NoError(err)
			s.NotEmpty(resp.GetResult())
			s.Len(resp.GetResult(), len(tt.req.GetUrls()))
		})
	}
}

func (s *ShortURLSuite) TestDeleteUserURLs() {
	ids := []string{"1", "2", "3"}
	visitorUUID := gofakeit.UUID()

	s.mockURLProvider.EXPECT().MarkAsDeleted(gomock.Any(), ids, visitorUUID).Return(nil)

	md := metadata.New(map[string]string{string(itrcpt.VisitorUUIDKey): visitorUUID})
	mdCtx := metadata.NewOutgoingContext(s.T().Context(), md)

	_, err := s.grpcClient.Client.DeleteUserURLs(mdCtx, &pb.DeleteUserURLsRequest{
		Ids: ids,
	})

	s.Require().NoError(err)
}

func (s *ShortURLSuite) serverUp() *bufconn.Listener {
	buffer := 1024 * 1024
	server := grpc.NewServer(grpc.UnaryInterceptor(itrcpt.Visitor))
	pb.RegisterShortURLServer(server, s.handler)
	lis := bufconn.Listen(buffer)
	go func() {
		if err := server.Serve(lis); err != nil {
			s.T().Logf("start server error: %v", err)
		}
	}()
	s.server = server
	return lis
}
