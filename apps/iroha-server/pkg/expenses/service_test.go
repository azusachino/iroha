package expenses

import (
	"errors"
	"testing"
	"time"
)

func TestNormalizeCreate(t *testing.T) {
	amount := int64(800)
	normalized, fingerprint, err := NormalizeCreate(CreateInput{
		OccurredOn: time.Date(2026, time.August, 12, 23, 45, 0, 0, time.FixedZone("JST", 9*60*60)),
		Currency:   " jpy ", AmountMinor: 1300, Category: " FOOD ", Merchant: " Ramen Shop ", Note: " Lunch ",
		Items: []Item{{Name: " Ramen ", AmountMinor: &amount}}, Source: Source{Kind: " LOCAL_AGENT ", Ref: " receipt-1 "},
	})
	if err != nil {
		t.Fatalf("NormalizeCreate() error = %v", err)
	}
	if normalized.OccurredOn.Location() != time.UTC || normalized.OccurredOn.Format("2006-01-02") != "2026-08-12" {
		t.Fatalf("OccurredOn = %v, want UTC calendar date", normalized.OccurredOn)
	}
	if normalized.Currency != "JPY" || normalized.Category != "food" || normalized.Merchant != "Ramen Shop" || normalized.Source.Kind != "local_agent" {
		t.Fatalf("normalized = %+v", normalized)
	}
	if len(normalized.Items) != 1 || normalized.Items[0].Name != "Ramen" || fingerprint == "" {
		t.Fatalf("normalized items/fingerprint = %+v/%q", normalized.Items, fingerprint)
	}
}

func TestNormalizeCreateFingerprintIsStableAndContentSensitive(t *testing.T) {
	base := CreateInput{
		OccurredOn: time.Date(2026, time.August, 12, 0, 0, 0, 0, time.UTC), Currency: "JPY", AmountMinor: 800,
		Category: "food", Source: Source{Kind: "cli", Ref: "receipt-1"},
	}
	_, first, err := NormalizeCreate(base)
	if err != nil {
		t.Fatal(err)
	}
	_, second, err := NormalizeCreate(base)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("same input fingerprints differ: %q != %q", first, second)
	}
	base.AmountMinor++
	_, changed, err := NormalizeCreate(base)
	if err != nil {
		t.Fatal(err)
	}
	if first == changed {
		t.Fatal("changed input retained the same fingerprint")
	}
}

func TestNormalizeRejectsInvalidValues(t *testing.T) {
	base := CreateInput{
		OccurredOn: time.Date(2026, time.August, 12, 0, 0, 0, 0, time.UTC), Currency: "JPY", AmountMinor: 1,
		Category: "food", Source: Source{Kind: "cli", Ref: "receipt-1"},
	}
	tests := []struct {
		name string
		edit func(*CreateInput)
	}{
		{name: "zero date", edit: func(input *CreateInput) { input.OccurredOn = time.Time{} }},
		{name: "currency", edit: func(input *CreateInput) { input.Currency = "CNY" }},
		{name: "amount", edit: func(input *CreateInput) { input.AmountMinor = 0 }},
		{name: "category", edit: func(input *CreateInput) { input.Category = "travel" }},
		{name: "source path", edit: func(input *CreateInput) { input.Source.Ref = "/tmp/receipt.jpg" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := base
			test.edit(&input)
			if _, _, err := NormalizeCreate(input); !errors.Is(err, ErrInvalidExpense) {
				t.Fatalf("error = %v, want ErrInvalidExpense", err)
			}
		})
	}
}
