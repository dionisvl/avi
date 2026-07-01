package model_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dionisvl/avi/api-go/internal/model"
)

func TestNewItemCondition(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      model.ItemCondition
		wantError bool
	}{
		{"new", "new", model.ItemConditionNew, false},
		{"used", "used", model.ItemConditionUsed, false},
		{"invalid", "broken", "", true},
		{"empty", "", "", true},
		{"case sensitive uppercase", "NEW", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := model.NewItemCondition(tt.input)
			if tt.wantError {
				assert.Error(t, err)
				assert.Equal(t, model.ErrInvalidItemCondition, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
