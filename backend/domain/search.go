package domain

type MaterialCandidate struct {
	ID                         int64   `json:"id"`
	Name                       string  `json:"name"`
	Formula                    string  `json:"formula,omitempty"`
	Category                   string  `json:"category"`
	Subcategory                string  `json:"subcategory,omitempty"`
	Density                    float64 `json:"density"`
	GlassTransitionTemp        float64 `json:"glass_transition_temp"`
	HeatDeflectionTemp         float64 `json:"heat_deflection_temp"`
	MeltingPoint               float64 `json:"melting_point"`
	BoilingPoint               float64 `json:"boiling_point"`
	ThermalConductivity        float64 `json:"thermal_conductivity"`
	SpecificHeat               float64 `json:"specific_heat"`
	ThermalExpansion           float64 `json:"thermal_expansion"`
	ElectricalResistivity      float64 `json:"electrical_resistivity"`
	YieldStrength              float64 `json:"yield_strength"`
	TensileStrength            float64 `json:"tensile_strength"`
	YoungsModulus              float64 `json:"youngs_modulus"`
	HardnessVickers            float64 `json:"hardness_vickers"`
	PoissonsRatio              float64 `json:"poissons_ratio"`
	ProcessingTempMin          float64 `json:"processing_temp_min_c"`
	ProcessingTempMax          float64 `json:"processing_temp_max_c"`
	Crystallinity              float64 `json:"crystallinity"`
	CrystalSystem              string  `json:"crystal_system"`
	FractureToughness          float64 `json:"fracture_toughness"`
	WeibullModulus             float64 `json:"weibull_modulus"`
	InterlaminarShearStrength  float64 `json:"interlaminar_shear_strength"`
	FiberVolumeFraction        float64 `json:"fiber_volume_fraction"`
	Source                     string  `json:"source"`
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
	RecommendedMaterial string              `json:"recommended_material"`
	WhyItMatches        []string            `json:"why_it_matches"`
	TradeOffs           []string            `json:"trade_offs"`
	Confidence          string              `json:"confidence"`
	ConfidenceScore     float64             `json:"confidence_score"`
	Sources             []int64             `json:"sources"`
	Report              string              `json:"report"`
	Candidates          []MaterialCandidate `json:"candidates,omitempty"`
}
