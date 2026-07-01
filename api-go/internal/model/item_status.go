package model

import "slices"

type ItemStatus string

const (
	ItemStatusPublished ItemStatus = "published"
	ItemStatusDraft     ItemStatus = "draft"
	ItemStatusArchived  ItemStatus = "archived"
	ItemStatusSold      ItemStatus = "sold"
)

func (s ItemStatus) IsPublished() bool {
	return s == ItemStatusPublished
}

var itemStatusTransitions = map[ItemStatus][]ItemStatus{
	ItemStatusDraft:     {ItemStatusPublished, ItemStatusArchived},
	ItemStatusPublished: {ItemStatusDraft, ItemStatusArchived, ItemStatusSold},
	ItemStatusSold:      {ItemStatusArchived, ItemStatusPublished},
	ItemStatusArchived:  {ItemStatusPublished, ItemStatusDraft},
}

func (s ItemStatus) CanTransitionTo(next ItemStatus) bool {
	if s == next {
		return true
	}
	return slices.Contains(itemStatusTransitions[s], next)
}
