package domain

type MaterialCandidate struct {
	ID            int64
	Name          string
	Category      string
	YieldStrength float64
}

type NumericFilter struct {
	Field   string   `json:"field"`
	Minimum *float64 `json:"minimum,omitempty"`
	Maximum *float64 `json:"maximum,omitempty"`
}

type SearchIntent struct {
	Domain                  string   `json:"domain,omitempty"`
	MinYieldStrength        *float64 `json:"min_yield_strength,omitempty"`
	MinOperatingTemperature *float64 `json:"min_operating_temperature,omitempty"`
	MaxDensity              *float64 `json:"max_density,omitempty"`
	BudgetConstraint        string   `json:"budget_constraint,omitempty"`
}

type StructuredRecommendation struct {
	RecommendedMaterial string   `json:"recommended_material"`
	WhyItMatches        []string `json:"why_it_matches"`
	TradeOffs           []string `json:"trade_offs"`
	Confidence          string   `json:"confidence"`
	ConfidenceScore     float64  `json:"confidence_score"`
	Sources             []int64  `json:"sources"`
	Report              string   `json:"report"`
}
