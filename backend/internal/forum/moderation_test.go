package forum

import (
	"errors"
	"strings"
	"testing"
)

func TestModerationReasonBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name   string
		reason string
		valid  bool
	}{
		{"trimmed", " \n Нарушение правил \t", true},
		{"unicode_limit", strings.Repeat("я", 1000), true},
		{"unicode_over_limit", strings.Repeat("я", 1001), false},
		{"whitespace", " \n\t ", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			decision := reportDecisionInput{Decision: "hide", Reason: tc.reason}
			status := topicStatusInput{Status: "closed", Reason: tc.reason}
			for _, err := range []error{decision.validate(), status.validate()} {
				if tc.valid {
					if err != nil {
						t.Fatal(err)
					}
				} else {
					assertModerationValidationError(t, err)
				}
			}
			want := strings.TrimSpace(tc.reason)
			if decision.Reason != want || status.Reason != want {
				t.Fatal("reason must be trimmed before it is persisted in the audit")
			}
		})
	}
	for _, decision := range []string{"delete", "Hide", " hide "} {
		in := reportDecisionInput{Decision: decision, Reason: "Причина"}
		assertModerationValidationError(t, in.validate())
	}
	for _, status := range []string{"hidden", "Closed", " closed "} {
		in := topicStatusInput{Status: status, Reason: "Причина"}
		assertModerationValidationError(t, in.validate())
	}
}

func TestSectionInputBoundaries(t *testing.T) {
	in := sectionInput{Slug: "  test-section\n", Title: " " + strings.Repeat("я", 80) + " ", Description: strings.Repeat("ю", 240), Archived: true}
	if err := validateSection(&in); err != nil {
		t.Fatal(err)
	}
	if in.Slug != "test-section" || in.Title != strings.Repeat("я", 80) || !in.Archived {
		t.Fatal("normalization changed the section data")
	}
	for _, tc := range []struct {
		name string
		in   sectionInput
	}{
		{"title_over_limit", sectionInput{Slug: "valid", Title: strings.Repeat("я", 81)}},
		{"description_over_limit", sectionInput{Slug: "valid", Title: "Раздел", Description: strings.Repeat("ю", 241)}},
		{"empty_title", sectionInput{Slug: "valid", Title: "   "}},
		{"non_latin_slug", sectionInput{Slug: "раздел", Title: "Раздел"}},
		{"uppercase_slug", sectionInput{Slug: "Section", Title: "Раздел"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assertModerationValidationError(t, validateSection(&tc.in))
		})
	}
}

func assertModerationValidationError(t *testing.T, err error) {
	t.Helper()
	var p *apiError
	if !errors.As(err, &p) || p.Status != 400 || p.Code != "validation" {
		t.Fatalf("expected validation error, got %v", err)
	}
}
