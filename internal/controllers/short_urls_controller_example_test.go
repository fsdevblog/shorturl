package controllers

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/fsdevblog/shorturl/internal/config"
	"github.com/fsdevblog/shorturl/internal/controllers/mocksctrl"
	"github.com/fsdevblog/shorturl/internal/logs"
	"github.com/fsdevblog/shorturl/internal/models"
	"github.com/golang/mock/gomock"
)

type mockTestHelper struct{}

func (h *mockTestHelper) Errorf(_ string, _ ...interface{}) {}
func (h *mockTestHelper) Fatalf(_ string, _ ...interface{}) {}

// ExampleShortURLController_CreateShortURL тест на создание ссылки.
func ExampleShortURLController_CreateShortURL() {
	h := new(mockTestHelper)
	// Настраиваем тестовое окружение
	ctrl := gomock.NewController(h)
	defer ctrl.Finish()
	mockStore := mocksctrl.NewMockShortURLStore(ctrl)

	// Настраиваем роутер
	router := SetupRouter(RouterParams{
		URLService:  mockStore,
		PingService: nil,
		AppConf: config.Config{
			ServerAddress:    ":80",
			BaseURL:          "http://test.com",
			VisitorJWTSecret: "secret",
		},
		Logger: logs.MustNew(func(o *logs.LoggerOptions) {
			o.Level = logs.LevelTypeError
		}),
	})

	testingURL := "https://example.com"
	mockStore.EXPECT().
		Create(gomock.Any(), gomock.Any(), testingURL).
		Return(&models.URL{
			URL:             testingURL,
			ShortIdentifier: "123123",
		}, true, nil).Times(1)

	// Готовим запрос
	jsonStr := fmt.Sprintf(`{"url":"%s"}`, testingURL)
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewBufferString(jsonStr))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Выполняем запрос
	router.ServeHTTP(w, req)

	// Выводим результат
	fmt.Printf("Status: %d\n", w.Code)
	fmt.Printf("Response: %s\n", w.Body.String())

	// Output:
	// Status: 201
	// Response: {"result":"http://test.com/123123"}
}
