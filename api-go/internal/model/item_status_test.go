package model_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dionisvl/avi/api-go/internal/model"
)

func TestItemStatusIsPublished(t *testing.T) {
	tests := []struct {
		name   string
		status model.ItemStatus
		want   bool
	}{
		{name: "published", status: model.ItemStatusPublished, want: true},
		{name: "draft", status: model.ItemStatusDraft, want: false},
		{name: "archived", status: model.ItemStatusArchived, want: false},
		{name: "sold", status: model.ItemStatusSold, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.status.IsPublished()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestItemStatusCanTransitionTo(t *testing.T) {
	tests := []struct {
		name      string
		from      model.ItemStatus
		to        model.ItemStatus
		wantAllow bool
	}{
		{"draft to published", model.ItemStatusDraft, model.ItemStatusPublished, true},
		{"draft to archived", model.ItemStatusDraft, model.ItemStatusArchived, true},
		{"draft to sold", model.ItemStatusDraft, model.ItemStatusSold, false},
		{"draft to draft", model.ItemStatusDraft, model.ItemStatusDraft, true},

		{"published to draft", model.ItemStatusPublished, model.ItemStatusDraft, true},
		{"published to archived", model.ItemStatusPublished, model.ItemStatusArchived, true},
		{"published to sold", model.ItemStatusPublished, model.ItemStatusSold, true},
		{"published to published", model.ItemStatusPublished, model.ItemStatusPublished, true},

		{"sold to archived", model.ItemStatusSold, model.ItemStatusArchived, true},
		{"sold to published", model.ItemStatusSold, model.ItemStatusPublished, true},
		{"sold to draft", model.ItemStatusSold, model.ItemStatusDraft, false},

		{"archived to published", model.ItemStatusArchived, model.ItemStatusPublished, true},
		{"archived to draft", model.ItemStatusArchived, model.ItemStatusDraft, true},
		{"archived to sold", model.ItemStatusArchived, model.ItemStatusSold, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.from.CanTransitionTo(tt.to)
			assert.Equal(t, tt.wantAllow, got)
		})
	}
}
