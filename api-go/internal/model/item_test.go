package model_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/dionisvl/avi/api-go/internal/model"
)

func TestNewItem(t *testing.T) {
	sellerID := uuid.New()
	createdByID := uuid.New()
	cityID := uuid.New()
	categoryID := uuid.New()
	thumbnailID := uuid.New()
	tags := []string{"electronics", "used"}

	input := model.NewItemInput{
		ID:          uuid.New(),
		SellerID:    sellerID,
		CreatedBy:   &createdByID,
		Slug:        "test-item",
		Title:       "Test Item",
		CategoryID:  categoryID,
		Description: "A test item",
		Tags:        &tags,
		CityID:      cityID,
		Condition:   model.ItemConditionUsed,
		Price:       nil,
		ThumbnailID: &thumbnailID,
	}

	item := model.NewItem(input)

	assert.Equal(t, input.ID, item.ID)
	assert.Equal(t, input.SellerID, item.SellerID)
	assert.Equal(t, input.CreatedBy, item.CreatedBy)
	assert.Equal(t, input.Slug, item.Slug)
	assert.Equal(t, input.Title, item.Title)
	assert.Equal(t, input.CategoryID, item.CategoryID)
	assert.Equal(t, input.Description, item.Description)
	assert.Equal(t, input.Tags, item.Tags)
	assert.Equal(t, input.Condition, item.Condition)
	assert.Equal(t, input.ThumbnailID, item.ThumbnailID)
	assert.Equal(t, model.ItemStatusPublished, item.Status)
	assert.Equal(t, input.CityID, item.City.ID)
	assert.True(t, item.CreatedAt.Before(time.Now().Add(time.Second)))
	assert.True(t, item.UpdatedAt.Before(time.Now().Add(time.Second)))
}

func TestItemWithDetailsCanBeManagedBy(t *testing.T) {
	sellerID := uuid.New()
	creatorID := uuid.New()
	otherID := uuid.New()

	tests := []struct {
		name      string
		item      *model.ItemWithDetails
		requestBy uuid.UUID
		want      bool
	}{
		{
			name: "seller can manage",
			item: &model.ItemWithDetails{
				Item: model.Item{
					SellerID: sellerID,
				},
				Seller: model.Seller{ID: sellerID},
			},
			requestBy: sellerID,
			want:      true,
		},
		{
			name: "creator can manage",
			item: &model.ItemWithDetails{
				Item: model.Item{
					SellerID:  sellerID,
					CreatedBy: &creatorID,
				},
				Seller: model.Seller{ID: sellerID},
			},
			requestBy: creatorID,
			want:      true,
		},
		{
			name: "unrelated user cannot manage",
			item: &model.ItemWithDetails{
				Item: model.Item{
					SellerID:  sellerID,
					CreatedBy: &creatorID,
				},
				Seller: model.Seller{ID: sellerID},
			},
			requestBy: otherID,
			want:      false,
		},
		{
			name: "nil CreatedBy, seller matches",
			item: &model.ItemWithDetails{
				Item: model.Item{
					SellerID:  sellerID,
					CreatedBy: nil,
				},
				Seller: model.Seller{ID: sellerID},
			},
			requestBy: sellerID,
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.item.CanBeManagedBy(tt.requestBy)
			assert.Equal(t, tt.want, got)
		})
	}
}
