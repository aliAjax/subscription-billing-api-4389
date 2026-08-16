package service

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"subscription-billing-api/internal/subscription/model"
	"subscription-billing-api/internal/subscription/repository"
)

func newTestService(t *testing.T) SubscriptionService {
	t.Helper()
	repo, err := repository.NewJSONRepository(filepath.Join(t.TempDir(), "subscriptions.json"))
	if err != nil {
		t.Fatalf("new repository: %v", err)
	}
	svc := New(repo)
	fixedNow := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	svc.(*subscriptionService).now = func() time.Time { return fixedNow }
	return svc
}

func createReq(name string, amount float64, status model.Status, daysFromNow int) model.CreateRequest {
	return model.CreateRequest{
		Name:            name,
		Amount:          amount,
		Status:          status,
		NextRenewalDate: time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC).AddDate(0, 0, daysFromNow).Format(model.DateLayout),
	}
}

func TestServiceCreateMetadataReady(t *testing.T) {
	svc := newTestService(t)
	req := createReq("Netflix", 39.9, "", 10)

	sub, err := svc.Create(req)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if sub.Status != model.StatusActive {
		t.Fatalf("expected active status, got %q", sub.Status)
	}
	if sub.Metadata == nil {
		t.Fatal("expected metadata map to be initialized")
	}
	if sub.Metadata["source"] != "api" {
		t.Fatalf("expected source metadata, got %#v", sub.Metadata)
	}
}

func TestServiceListFiltersAndSorts(t *testing.T) {
	svc := newTestService(t)
	if _, err := svc.Create(createReq("Later", 10, model.StatusActive, 10)); err != nil {
		t.Fatalf("create later: %v", err)
	}
	if _, err := svc.Create(createReq("Sooner", 10, model.StatusActive, 2)); err != nil {
		t.Fatalf("create sooner: %v", err)
	}
	if _, err := svc.Create(createReq("Paused", 10, model.StatusPaused, 5)); err != nil {
		t.Fatalf("create paused: %v", err)
	}

	items, err := svc.List("active")
	if err != nil {
		t.Fatalf("list active: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 active items, got %d", len(items))
	}
	if items[0].Name != "Sooner" || items[1].Name != "Later" {
		t.Fatalf("unexpected sort order: %#v", items)
	}

	if _, err := svc.List("invalid"); !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error for invalid status, got %v", err)
	}
}

func TestServiceListDoesNotMutateRepository(t *testing.T) {
	svc := newTestService(t)
	if _, err := svc.Create(createReq("PausedFirst", 10, model.StatusPaused, 5)); err != nil {
		t.Fatalf("create paused: %v", err)
	}
	if _, err := svc.Create(createReq("ActiveSecond", 10, model.StatusActive, 3)); err != nil {
		t.Fatalf("create active: %v", err)
	}

	if _, err := svc.List("active"); err != nil {
		t.Fatalf("filter list: %v", err)
	}
	all, err := svc.List("")
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 items after filtering, got %d: %#v", len(all), all)
	}
	seen := map[string]bool{}
	for _, item := range all {
		seen[item.Name] = true
	}
	if !seen["PausedFirst"] || !seen["ActiveSecond"] {
		t.Fatalf("repository items were lost after filtering: %#v", all)
	}
}

func TestServiceUpcomingChargesBoundary(t *testing.T) {
	svc := newTestService(t)
	for _, days := range []int{0, 6, 7} {
		name := "day"
		switch days {
		case 0:
			name = "today"
		case 6:
			name = "last-in-range"
		case 7:
			name = "first-out-of-range"
		}
		if _, err := svc.Create(createReq(name, 10, model.StatusActive, days)); err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
	}

	report, err := svc.UpcomingCharges(7)
	if err != nil {
		t.Fatalf("upcoming charges: %v", err)
	}
	if report.Count != 2 {
		t.Fatalf("expected 2 upcoming charges, got %d: %#v", report.Count, report.Subscriptions)
	}
	if report.EndDate != "2026-08-22" {
		t.Fatalf("expected inclusive 7-day end date 2026-08-22, got %s", report.EndDate)
	}
}

func TestServiceMonthlyExpectedSpendOnlyActive(t *testing.T) {
	svc := newTestService(t)
	if _, err := svc.Create(createReq("Active", 10, model.StatusActive, 5)); err != nil {
		t.Fatalf("create active: %v", err)
	}
	if _, err := svc.Create(createReq("Paused", 20, model.StatusPaused, 6)); err != nil {
		t.Fatalf("create paused: %v", err)
	}
	if _, err := svc.Create(createReq("Cancelled", 30, model.StatusCancelled, 7)); err != nil {
		t.Fatalf("create cancelled: %v", err)
	}

	report, err := svc.MonthlyExpectedSpend()
	if err != nil {
		t.Fatalf("monthly total: %v", err)
	}
	if report.Amount != 10 || report.Count != 1 {
		t.Fatalf("expected amount=10 count=1, got amount=%v count=%d", report.Amount, report.Count)
	}
}

func TestServiceUpdateRenewalDateNotFound(t *testing.T) {
	svc := newTestService(t)
	_, err := svc.UpdateRenewalDate("missing", model.UpdateRenewalDateRequest{
		NextRenewalDate: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC).Format(model.DateLayout),
	})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestServiceDeleteNotFound(t *testing.T) {
	svc := newTestService(t)
	if err := svc.Delete("missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
