package model

import (
	"math"
	"testing"
	"time"
)

func futureDate(t *testing.T) string {
	t.Helper()
	return time.Now().AddDate(0, 0, 14).Format(DateLayout)
}

func TestParseStatusRejectsInvalid(t *testing.T) {
	if _, err := ParseStatus("expired"); err == nil {
		t.Fatal("expected invalid status to return an error")
	}
}

func TestParseStatusEmptyDefaultsActive(t *testing.T) {
	got, err := ParseStatus("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != StatusActive {
		t.Fatalf("expected active, got %q", got)
	}
}

func TestNormalizeMetadataInitializesNil(t *testing.T) {
	got := NormalizeMetadata(nil)
	if got == nil {
		t.Fatal("expected non-nil map")
	}
}

func TestValidateCreateRequestNormalizesMetadataAndStatus(t *testing.T) {
	req := &CreateRequest{
		Name:            " Netflix ",
		Amount:          39.955,
		Status:          "",
		NextRenewalDate: futureDate(t),
		Metadata:        map[string]string{" Billing ": " Monthly "},
	}

	if err := ValidateCreateRequest(req); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
	if req.Status != StatusActive {
		t.Fatalf("expected active status, got %q", req.Status)
	}
	if req.Metadata["Billing"] != "Monthly" {
		t.Fatalf("metadata was not normalized: %#v", req.Metadata)
	}
	if got := RoundMoney(req.Amount); got != 39.96 {
		t.Fatalf("expected rounded amount 39.96, got %v", got)
	}
}

func TestValidateRenewalDateRejectsPastDate(t *testing.T) {
	past := time.Now().AddDate(0, 0, -1).Format(DateLayout)
	if err := ValidateRenewalDate(past); err == nil {
		t.Fatal("expected past date to be rejected")
	}
}

func TestValidateRenewalDateRejectsNonZeroPaddedDate(t *testing.T) {
	next := time.Now().AddDate(0, 0, 10)
	nonPadded := next.Format("2006-1-2")
	if err := ValidateRenewalDate(nonPadded); err == nil {
		t.Fatal("expected non-zero-padded date to be rejected")
	}
}

func TestRoundMoney(t *testing.T) {
	cases := map[float64]float64{
		39.955: 39.96,
		39.951: 39.95,
		12.345: 12.35,
		12.344: 12.34,
	}
	for input, want := range cases {
		if got := RoundMoney(input); math.Abs(got-want) > 0.000001 {
			t.Fatalf("RoundMoney(%v) = %v, want %v", input, got, want)
		}
	}
}
