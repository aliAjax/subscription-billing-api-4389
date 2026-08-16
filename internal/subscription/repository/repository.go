package repository

import (
	"errors"

	"subscription-billing-api/internal/subscription/model"
)

var ErrNotFound = errors.New("subscription not found")

type Repository interface {
	List() ([]model.Subscription, error)
	GetByID(id string) (model.Subscription, error)
	Create(subscription model.Subscription) error
	Update(subscription model.Subscription) error
	Delete(id string) error
}
