package model

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
)

const DateLayout = "2006-01-02"

type Status string

const (
	StatusActive    Status = "active"
	StatusPaused    Status = "paused"
	StatusCancelled Status = "cancelled"
)

func (s Status) Valid() bool {
	switch s {
	case StatusActive, StatusPaused, StatusCancelled:
		return true
	default:
		return false
	}
}

func ParseStatus(value string) (Status, error) {
	normalized := Status(strings.ToLower(strings.TrimSpace(value)))
	if normalized == "" {
		return StatusActive, nil
	}
	if !normalized.Valid() {
		return "", fmt.Errorf("status must be one of active, paused, cancelled: got %q", value)
	}
	return normalized, nil
}

func NormalizeMetadata(metadata map[string]string) map[string]string {
	if metadata == nil {
		return make(map[string]string)
	}
	normalized := make(map[string]string, len(metadata))
	for key, value := range metadata {
		normalized[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return normalized
}

type Subscription struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Description     string            `json:"description,omitempty"`
	Amount          float64           `json:"amount"`
	Status          Status            `json:"status"`
	NextRenewalDate string            `json:"next_renewal_date"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

type CreateRequest struct {
	Name            string            `json:"name"`
	Description     string            `json:"description,omitempty"`
	Amount          float64           `json:"amount"`
	Status          Status            `json:"status"`
	NextRenewalDate string            `json:"next_renewal_date"`
	Metadata        map[string]string `json:"metadata,omitempty"`
}

type UpdateRenewalDateRequest struct {
	NextRenewalDate string `json:"next_renewal_date"`
}

func NewID() (string, error) {
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}

	randomBytes[6] = (randomBytes[6] & 0x0f) | 0x40
	randomBytes[8] = (randomBytes[8] & 0x3f) | 0x80

	return fmt.Sprintf(
		"%x-%x-%x-%x-%x",
		randomBytes[0:4],
		randomBytes[4:6],
		randomBytes[6:8],
		randomBytes[8:10],
		randomBytes[10:16],
	), nil
}

func ValidateCreateRequest(req *CreateRequest) error {
	if req == nil {
		return errors.New("request body is required")
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return errors.New("name is required")
	}
	if len([]rune(req.Name)) > 100 {
		return errors.New("name must not exceed 100 characters")
	}

	req.Description = strings.TrimSpace(req.Description)
	if len([]rune(req.Description)) > 500 {
		return errors.New("description must not exceed 500 characters")
	}

	if math.IsNaN(req.Amount) || math.IsInf(req.Amount, 0) {
		return errors.New("amount must be a valid number")
	}
	if req.Amount <= 0 {
		return errors.New("amount must be greater than zero")
	}
	if req.Amount > 1_000_000 {
		return errors.New("amount must not exceed 1000000")
	}

	status, err := ParseStatus(string(req.Status))
	if err != nil {
		return err
	}
	req.Status = status

	req.Metadata = NormalizeMetadata(req.Metadata)

	if err := ValidateRenewalDate(req.NextRenewalDate); err != nil {
		return err
	}

	req.Amount = RoundMoney(req.Amount)
	return nil
}

func ValidateRenewalDate(value string) error {
	parsed, err := ParseDateStrict(value)
	if err != nil {
		return err
	}

	today, _ := time.Parse(DateLayout, time.Now().Format(DateLayout))
	if parsed.Before(today) {
		return errors.New("next_renewal_date must not be in the past")
	}
	return nil
}

func ParseDateStrict(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, errors.New("next_renewal_date is required and must use YYYY-MM-DD")
	}

	parsed, err := time.Parse(DateLayout, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("next_renewal_date must use YYYY-MM-DD: %w", err)
	}
	if parsed.Format(DateLayout) != value {
		return time.Time{}, errors.New("next_renewal_date must use zero-padded YYYY-MM-DD")
	}

	return parsed, nil
}

func RoundMoney(value float64) float64 {
	return math.Round(value*100) / 100
}
