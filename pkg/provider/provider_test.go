package provider

import (
	"testing"
)

type mockProviderImpl struct {
	slug string
	name string
}

func (m *mockProviderImpl) Slug() string        { return m.slug }
func (m *mockProviderImpl) DisplayName() string { return m.name }

type customFeature interface {
	DoFeature() string
}

type featureProvider struct {
	mockProviderImpl
}

func (f *featureProvider) DoFeature() string {
	return "feature-done"
}

func TestProviderRegistry(t *testing.T) {
	p1 := &mockProviderImpl{slug: "brevo", name: "Brevo"}
	p2 := &featureProvider{mockProviderImpl{slug: "didit", name: "Didit"}}

	RegisterProvider(p1, p2)

	// GetProviderImpl
	got, ok := GetProviderImpl("brevo")
	if !ok {
		t.Fatal("expected brevo to be found")
	}
	if got.DisplayName() != "Brevo" {
		t.Errorf("expected Brevo, got %s", got.DisplayName())
	}

	// Case-insensitive lookup
	got, ok = GetProviderImpl("BREVO")
	if !ok {
		t.Fatal("expected BREVO case-insensitive lookup to succeed")
	}
	if got.DisplayName() != "Brevo" {
		t.Errorf("expected Brevo, got %s", got.DisplayName())
	}

	// ConformsTo
	feat, ok := ConformsTo[customFeature](p2)
	if !ok {
		t.Fatal("expected p2 to conform to customFeature")
	}
	if feat.DoFeature() != "feature-done" {
		t.Errorf("unexpected feature result: %s", feat.DoFeature())
	}

	// GetImpl
	feat, ok = GetImpl[customFeature]("didit")
	if !ok {
		t.Fatal("expected didit to conform to customFeature via GetImpl")
	}
	if feat.DoFeature() != "feature-done" {
		t.Errorf("unexpected feature result: %s", feat.DoFeature())
	}

	// Non-conforming type
	_, ok = GetImpl[customFeature]("brevo")
	if ok {
		t.Error("expected brevo to not conform to customFeature")
	}

	// Missing provider
	_, ok = GetProviderImpl("unknown-provider")
	if ok {
		t.Error("expected unknown-provider to not be found")
	}
}
