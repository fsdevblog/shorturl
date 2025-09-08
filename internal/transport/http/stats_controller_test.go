package http

import (
	"net/http"
	"testing"

	"github.com/fsdevblog/shorturl/internal/transport/http/mocksctrl"
	"github.com/fsdevblog/shorturl/internal/transport/http/testutils"

	"github.com/fsdevblog/shorturl/internal/config"
	"github.com/fsdevblog/shorturl/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

type StatsControllerSuite struct {
	suite.Suite
	ctrl        *gomock.Controller
	mockService *mocksctrl.MockStatsProvider
	router      *gin.Engine
	config      *config.Config
}

func TestStatsController(t *testing.T) {
	suite.Run(t, new(StatsControllerSuite))
}

func (s *StatsControllerSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.mockService = mocksctrl.NewMockStatsProvider(s.ctrl)
	s.config = &config.Config{
		ServerAddress:    ":80",
		BaseURL:          "http://test.com:8080",
		VisitorJWTSecret: jwtSecret,
		TrustedSubnet:    "5.181.188.177/24",
	}
	s.router = SetupRouter(RouterParams{
		StatsService: s.mockService,
		AppConf:      s.config,
		Logger:       zap.NewNop(),
	})
}

func (s *StatsControllerSuite) TearDownTest() {
	s.ctrl.Finish()
}

func (s *StatsControllerSuite) Test_GetStats() {
	allowedIP := "5.181.188.177"

	expectedStats := services.Stats{
		URLs:  10,
		Users: 100,
	}
	// подготавливаем мок сервиса
	s.mockService.EXPECT().
		GetStats(gomock.Any()).
		Return(&expectedStats, nil).
		AnyTimes()

	s.Run("with allowed IP", func() {
		stats, response, errRequest := testutils.RequestWithTypedReturn[*StatsResponse](s.router, testutils.Params{
			Method: http.MethodGet,
			URL:    "/api/internal/stats",
		}, func(o *testutils.Options) {
			o.Headers = map[string]string{
				"X-Real-IP": allowedIP,
			}
		})
		s.Require().NoError(errRequest)
		errClose := response.Body.Close()
		s.Require().NoError(errClose)
		s.Require().Equal(http.StatusOK, response.StatusCode)
		s.Equal(expectedStats.URLs, stats.URLs)
		s.Equal(expectedStats.Users, stats.Users)
	})

	s.Run("with not allowed IP", func() {
		response, errRequest := testutils.Request(s.router, testutils.Params{
			Method: http.MethodGet,
			URL:    "/api/internal/stats",
		}, func(o *testutils.Options) {
			o.Headers = map[string]string{
				"X-Real-IP": "127.0.0.1",
			}
		})
		s.Require().NoError(errRequest)
		errClose := response.Body.Close()
		s.Require().NoError(errClose)
		s.Require().Equal(http.StatusForbidden, response.StatusCode)
	})

	s.Run("with empty config option trusted_subnet", func() {
		s.config.TrustedSubnet = ""
		s.router = SetupRouter(RouterParams{
			StatsService: s.mockService,
			AppConf:      s.config,
			Logger:       zap.NewNop(),
		})
		response, errRequest := testutils.Request(s.router, testutils.Params{
			Method: http.MethodGet,
			URL:    "/api/internal/stats",
		}, func(o *testutils.Options) {
			o.Headers = map[string]string{
				"X-Real-IP": allowedIP,
			}
		})
		s.Require().NoError(errRequest)
		errClose := response.Body.Close()
		s.Require().NoError(errClose)
		s.Require().Equal(http.StatusForbidden, response.StatusCode)
	})
}
