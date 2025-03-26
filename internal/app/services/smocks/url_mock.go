package smocks

import (
	"github.com/fsdevblog/shorturl/internal/app/models"
	"github.com/stretchr/testify/mock"
)

type URLMock struct {
	mock.Mock
}

func (u *URLMock) Create(rawURL string) (*models.URL, error) {
	args := u.Called(rawURL)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck,errcheck
	}
	return args.Get(0).(*models.URL), args.Error(1) //nolint:wrapcheck,errcheck
}

func (u *URLMock) GetByShortIdentifier(shortID string) (*models.URL, error) {
	args := u.Called(shortID)
	if args.Get(0) == nil {
		return nil, args.Error(1) //nolint:wrapcheck,errcheck
	}
	return args.Get(0).(*models.URL), args.Error(1) //nolint:wrapcheck,errcheck
}
