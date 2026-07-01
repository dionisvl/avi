package model_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/dionisvl/avi/api-go/internal/model"
)

func TestCategoryLocalizedName(t *testing.T) {
	tests := []struct {
		name     string
		category *model.Category
		locale   string
		want     string
	}{
		{
			name: "exact locale match",
			category: &model.Category{
				Names: map[string]string{
					"en": "Electronics",
					"ru": "Электроника",
				},
			},
			locale: "ru",
			want:   "Электроника",
		},
		{
			name: "fallback to English",
			category: &model.Category{
				Names: map[string]string{
					"en": "Electronics",
					"fr": "Électronique",
				},
			},
			locale: "de",
			want:   "Electronics",
		},
		{
			name: "no English fallback, use any",
			category: &model.Category{
				Names: map[string]string{
					"fr": "Électronique",
				},
			},
			locale: "de",
			want:   "Électronique",
		},
		{
			name:     "nil category",
			category: nil,
			locale:   "en",
			want:     "",
		},
		{
			name: "empty names",
			category: &model.Category{
				ID:    uuid.New(),
				Names: map[string]string{},
			},
			locale: "en",
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.category.LocalizedName(tt.locale)
			assert.Equal(t, tt.want, got)
		})
	}
}
