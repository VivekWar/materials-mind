package handlers

import (
	"strings"
	"testing"
)

func TestParseStructuredRecommendation_WithCodeFence(t *testing.T) {
	raw := "```json\n{\n" +
		"  \"recommended_material\": \"Ti-6Al-4V\",\n" +
		"  \"why_it_matches\": [\"high strength-to-weight\", \"good fatigue\"],\n" +
		"  \"trade_offs\": [\"higher cost\"],\n" +
		"  \"confidence\": \"High\",\n" +
		"  \"confidence_score\": 0.86,\n" +
		"  \"sources\": [7, 3],\n" +
		"  \"report\": \"Best overall match for aerospace load case.\"\n" +
		"}\n```"

	rec, err := parseStructuredRecommendation(raw)
	if err != nil {
		t.Fatalf("expected parse success, got error: %v", err)
	}

	if rec.RecommendedMaterial != "Ti-6Al-4V" {
		t.Fatalf("unexpected recommended material: %s", rec.RecommendedMaterial)
	}
	if rec.ConfidenceScore != 0.86 {
		t.Fatalf("unexpected confidence_score: %v", rec.ConfidenceScore)
	}
	if len(rec.Sources) != 2 {
		t.Fatalf("unexpected sources count: %d", len(rec.Sources))
	}
}

func TestParseStructuredRecommendation_InvalidJSON(t *testing.T) {
	_, err := parseStructuredRecommendation("{not-json")
	if err == nil {
		t.Fatal("expected parse error, got nil")
	}
}

func TestValidateRecommendation_SuccessNormalizesFields(t *testing.T) {
	candidates := []materialCandidate{
		{ID: 4, Name: "A", Category: "X", YieldStrength: 100},
		{ID: 2, Name: "B", Category: "Y", YieldStrength: 200},
	}

	rec := &structuredRecommendation{
		RecommendedMaterial: "Material-B",
		WhyItMatches:        []string{"meets strength", "good processing"},
		TradeOffs:           []string{"higher cost"},
		Confidence:          "medium",
		ConfidenceScore:     0.72,
		Sources:             []int64{4, 2, 4},
		Report:              "Recommended due to balanced strength and manufacturability.",
	}

	err := validateRecommendation(rec, candidates)
	if err != nil {
		t.Fatalf("expected validation success, got error: %v", err)
	}

	if rec.Confidence != "Medium" {
		t.Fatalf("expected confidence to normalize to Medium, got: %s", rec.Confidence)
	}

	if len(rec.Sources) != 2 || rec.Sources[0] != 2 || rec.Sources[1] != 4 {
		t.Fatalf("expected unique sorted sources [2 4], got: %v", rec.Sources)
	}
}

func TestValidateRecommendation_FailsForOutOfRangeConfidenceScore(t *testing.T) {
	candidates := []materialCandidate{{ID: 10, Name: "A", Category: "X", YieldStrength: 100}}
	rec := &structuredRecommendation{
		RecommendedMaterial: "Material-A",
		WhyItMatches:        []string{"match"},
		TradeOffs:           []string{"trade"},
		Confidence:          "High",
		ConfidenceScore:     1.5,
		Sources:             []int64{10},
		Report:              "Short report.",
	}

	err := validateRecommendation(rec, candidates)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if !strings.Contains(err.Error(), "confidence_score") {
		t.Fatalf("expected confidence_score error, got: %v", err)
	}
}

func TestValidateRecommendation_FailsForUnknownSource(t *testing.T) {
	candidates := []materialCandidate{{ID: 10, Name: "A", Category: "X", YieldStrength: 100}}
	rec := &structuredRecommendation{
		RecommendedMaterial: "Material-A",
		WhyItMatches:        []string{"match"},
		TradeOffs:           []string{"trade"},
		Confidence:          "Low",
		ConfidenceScore:     0.21,
		Sources:             []int64{999},
		Report:              "Short report.",
	}

	err := validateRecommendation(rec, candidates)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if !strings.Contains(err.Error(), "not in retrieved candidates") {
		t.Fatalf("expected source-id error, got: %v", err)
	}
}

func TestNormalizeSearchQuery(t *testing.T) {
	out := normalizeSearchQuery("  titanium   frame\n\tfor   drone ")
	if out != "titanium frame for drone" {
		t.Fatalf("unexpected normalized query: %q", out)
	}
}
