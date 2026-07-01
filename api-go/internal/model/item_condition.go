package model

import "errors"

var ErrInvalidItemCondition = errors.New("invalid item condition")

type ItemCondition string

const (
	ItemConditionNew  ItemCondition = "new"
	ItemConditionUsed ItemCondition = "used"
)

func NewItemCondition(v string) (ItemCondition, error) {
	switch ItemCondition(v) {
	case ItemConditionNew, ItemConditionUsed:
		return ItemCondition(v), nil
	}
	return "", ErrInvalidItemCondition
}
