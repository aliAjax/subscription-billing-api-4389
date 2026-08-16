package service

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"subscription-billing-api/internal/subscription/model"
	"subscription-billing-api/internal/subscription/repository"
)

type SubscriptionService interface {
	Create(req model.CreateRequest) (model.Subscription, error)
	List(status string) ([]model.Subscription, error)
	Get(id string) (model.Subscription, error)
	UpdateRenewalDate(id string, req model.UpdateRenewalDateRequest) (model.Subscription, error)
	Delete(id string) error
	MonthlyExpectedSpend() (MonthlyExpectedSpend, error)
	UpcomingCharges(days int) (UpcomingCharges, error)
}

type MonthlyExpectedSpend struct {
	Month    string  `json:"month"`
	Currency string  `json:"currency"`
	Amount   float64 `json:"amount"`
	Count    int     `json:"count"`
}

type UpcomingCharges struct {
	Days          int                  `json:"days"`
	StartDate     string               `json:"start_date"`
	EndDate       string               `json:"end_date"`
	Count         int                  `json:"count"`
	Subscriptions []model.Subscription `json:"subscriptions"`
}

type subscriptionService struct {
	repo repository.Repository
	now  func() time.Time
}

func New(repo repository.Repository) SubscriptionService {
	return &subscriptionService{
		repo: repo,
		now:  time.Now,
	}
}

func (s *subscriptionService) Create(req model.CreateRequest) (model.Subscription, error) {
	if err := model.ValidateCreateRequest(&req); err != nil {
		return model.Subscription{}, fmt.Errorf("%v: %v", ErrValidation, err)
	}

	id, err := model.NewID()
	if err != nil {
		return model.Subscription{}, fmt.Errorf("generate subscription id: %w", err)
	}

	now := s.now().UTC()
	subscription := model.Subscription{
		ID:              id,
		Name:            req.Name,
		Description:     req.Description,
		Amount:          req.Amount,
		Status:          req.Status,
		NextRenewalDate: req.NextRenewalDate,
		Metadata:        req.Metadata,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	subscription.Metadata["source"] = "api"

	if err := s.repo.Create(subscription); err != nil {
		return model.Subscription{}, fmt.Errorf("create subscription: %w", err)
	}

	return subscription, nil
}

func (s *subscriptionService) List(status string) ([]model.Subscription, error) {
	status = strings.TrimSpace(status)
	var statusFilter model.Status
	if status != "" && !strings.EqualFold(status, "all") {
		parsed, err := model.ParseStatus(status)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrValidation, err)
		}
		statusFilter = parsed
	}

	subscriptions, err := s.repo.List()
	if err != nil {
		return nil, err
	}

	if statusFilter != "" {
		filtered := subscriptions[:0]
		for _, subscription := range subscriptions {
			if subscription.Status == statusFilter {
				filtered = append(filtered, subscription)
			}
		}
		subscriptions = filtered
	}

	sortSubscriptions(subscriptions)
	return subscriptions, nil
}

func (s *subscriptionService) Get(id string) (model.Subscription, error) {
	subscription, err := s.repo.GetByID(id)
	if errors.Is(err, repository.ErrNotFound) {
		return model.Subscription{}, ErrNotFound
	}
	if err != nil {
		return model.Subscription{}, err
	}
	return subscription, nil
}

func (s *subscriptionService) UpdateRenewalDate(id string, req model.UpdateRenewalDateRequest) (model.Subscription, error) {
	req.NextRenewalDate = strings.TrimSpace(req.NextRenewalDate)
	if err := model.ValidateRenewalDate(req.NextRenewalDate); err != nil {
		return model.Subscription{}, fmt.Errorf("%w: %v", ErrValidation, err)
	}

	subscription, err := s.repo.GetByID(id)
	if errors.Is(err, repository.ErrNotFound) {
		return model.Subscription{}, ErrNotFound
	}
	if err != nil {
		return model.Subscription{}, err
	}

	subscription.NextRenewalDate = req.NextRenewalDate
	subscription.UpdatedAt = s.now().UTC()
	if err := s.repo.Update(subscription); err != nil {
		return model.Subscription{}, err
	}

	return subscription, nil
}

func (s *subscriptionService) Delete(id string) error {
	if err := s.repo.Delete(id); errors.Is(err, repository.ErrNotFound) {
		return ErrNotFound
	} else if err != nil {
		return err
	}
	return nil
}

func (s *subscriptionService) MonthlyExpectedSpend() (MonthlyExpectedSpend, error) {
	subscriptions, err := s.repo.List()
	if err != nil {
		return MonthlyExpectedSpend{}, err
	}

	var total float64
	var count int
	for _, subscription := range subscriptions {
		if subscription.Status == model.StatusActive {
			total += subscription.Amount
			count++
		}
	}

	return MonthlyExpectedSpend{
		Month:    s.now().Format("2006-01"),
		Currency: "CNY",
		Amount:   model.RoundMoney(total),
		Count:    count,
	}, nil
}

func (s *subscriptionService) UpcomingCharges(days int) (UpcomingCharges, error) {
	if days <= 0 {
		days = 7
	}
	if days > 365 {
		return UpcomingCharges{}, fmt.Errorf("%w: days must be between 1 and 365", ErrValidation)
	}

	subscriptions, err := s.repo.List()
	if err != nil {
		return UpcomingCharges{}, err
	}

	today, _ := time.Parse(model.DateLayout, s.now().Format(model.DateLayout))
	end := today.AddDate(0, 0, days-1)

	var upcoming []model.Subscription
	for _, subscription := range subscriptions {
		if subscription.Status != model.StatusActive {
			continue
		}

		renewalDate, err := model.ParseDateStrict(subscription.NextRenewalDate)
		if err != nil {
			return UpcomingCharges{}, fmt.Errorf("stored next_renewal_date for subscription %s is invalid: %w", subscription.ID, err)
		}

		if !renewalDate.Before(today) && !renewalDate.After(end) {
			upcoming = append(upcoming, subscription)
		}
	}

	sortSubscriptions(upcoming)
	return UpcomingCharges{
		Days:          days,
		StartDate:     today.Format(model.DateLayout),
		EndDate:       end.Format(model.DateLayout),
		Count:         len(upcoming),
		Subscriptions: upcoming,
	}, nil
}

func sortSubscriptions(subscriptions []model.Subscription) {
	sort.SliceStable(subscriptions, func(i, j int) bool {
		leftDate, _ := model.ParseDateStrict(subscriptions[i].NextRenewalDate)
		rightDate, _ := model.ParseDateStrict(subscriptions[j].NextRenewalDate)
		if !leftDate.Equal(rightDate) {
			return leftDate.Before(rightDate)
		}
		return strings.ToLower(subscriptions[i].Name) < strings.ToLower(subscriptions[j].Name)
	})
}
