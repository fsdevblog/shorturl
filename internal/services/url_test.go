package services

import (
	"context"
	"fmt"
	"testing"

	"github.com/brianvoe/gofakeit/v7"

	"github.com/fsdevblog/shorturl/internal/models"
	"github.com/fsdevblog/shorturl/internal/repositories"
	"github.com/fsdevblog/shorturl/internal/services/mocks"

	"github.com/golang/mock/gomock"
)

// BenchmarkURLService_BatchCreate_Different_Sizes тестирует производительность с разными размерами пакетов.
func BenchmarkURLService_BatchCreate_Different_Sizes(b *testing.B) {
	sizes := []int{1, 10, 100, 1000}
	ctrl := gomock.NewController(b)
	defer ctrl.Finish()
	for _, size := range sizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			mockRepo := mocks.NewMockURLRepository(ctrl)
			service := NewURLService(mockRepo)

			// Генерируем URLs нужного размера
			urls := make([]string, size)
			expectedResults := make([]repositories.BatchResult[models.URL], size)
			for i := range size {
				urls[i] = gofakeit.URL()
				expectedResults[i] = repositories.BatchResult[models.URL]{
					Value: models.URL{
						URL:             urls[i],
						ShortIdentifier: fmt.Sprintf("short%d", i),
					},
				}
			}

			expectedResult := &repositories.BatchCreateShortURLsResult{
				Results: expectedResults,
			}

			mockRepo.EXPECT().
				BatchCreate(gomock.Any(), gomock.Any()).
				Return(expectedResult, nil).
				AnyTimes()

			ctx := context.Background()
			visitorUUID := "test-visitor-uuid"

			b.ResetTimer()

			for range b.N {
				_, err := service.BatchCreate(ctx, visitorUUID, urls)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
