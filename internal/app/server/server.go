package server

import (
	"net/http"

	"github.com/fsdevblog/shorturl/internal/app/server/controllers"
	"github.com/fsdevblog/shorturl/internal/app/services"
	"github.com/sirupsen/logrus"
)

// Start Стартует веб сервер.
func Start(urlService services.IURLService) {
	mux := http.NewServeMux()

	shortURLController := controllers.NewShortURLController(urlService)

	mux.HandleFunc("/", shortURLController.Handler)

	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		logrus.WithError(err).Fatal("failed to start server")
	}
}
