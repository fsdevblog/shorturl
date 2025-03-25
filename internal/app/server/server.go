package server

import (
	"net/http"
	"time"

	"github.com/fsdevblog/shorturl/internal/app/server/controllers"
	"github.com/fsdevblog/shorturl/internal/app/services"
	"github.com/sirupsen/logrus"
)

const (
	ReadTimeout       = 5 * time.Second
	WriteTimeout      = 10 * time.Second
	IdleTimeout       = 120 * time.Second
	ReadHeaderTimeout = 2 * time.Second
)

// Start Стартует веб сервер.
func Start(urlService services.IURLService) {
	mux := http.NewServeMux()

	shortURLController := controllers.NewShortURLController(urlService)

	mux.HandleFunc("/", shortURLController.Handler)

	s := http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadTimeout:       ReadTimeout,
		ReadHeaderTimeout: ReadHeaderTimeout,
		WriteTimeout:      WriteTimeout,
		IdleTimeout:       IdleTimeout,
	}
	err := s.ListenAndServe()
	if err != nil {
		logrus.WithError(err).Fatal("failed to start server")
	}
}
