package repositories

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/vivekwar/materialmind/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MaterialRepository interface {
	SearchByVector(ctx context.Context, embedding []float32, limit int, categoryFilter string) ([]domain.MaterialCandidate, error)
	SearchByIntent(ctx context.Context, intent domain.SearchIntent, query string, limit int) ([]domain.MaterialCandidate, error)
}

type materialRepository struct {
	pool *pgxpool.Pool
}

func NewMaterialRepository(pool *pgxpool.Pool) MaterialRepository {
	return &materialRepository{pool: pool}
}

const selectAllCols = `
	SELECT id, name, COALESCE(formula, ''), COALESCE(category, ''), COALESCE(subcategory, ''), 
	       COALESCE((specific_properties->>'density')::float, 0), COALESCE((specific_properties->>'glass_transition_temp')::float, 0), 
	       COALESCE((specific_properties->>'heat_deflection_temp')::float, 0), COALESCE((specific_properties->>'melting_point')::float, 0), 
	       COALESCE((specific_properties->>'boiling_point')::float, 0), COALESCE((specific_properties->>'thermal_conductivity')::float, 0), 
	       COALESCE((specific_properties->>'specific_heat')::float, 0), COALESCE((specific_properties->>'thermal_expansion')::float, 0), 
	       COALESCE((specific_properties->>'electrical_resistivity')::float, 0), COALESCE((specific_properties->>'yield_strength')::float, 0), 
	       COALESCE((specific_properties->>'tensile_strength')::float, 0), COALESCE((specific_properties->>'youngs_modulus')::float, 0), 
	       COALESCE((specific_properties->>'hardness_vickers')::float, 0), COALESCE((specific_properties->>'poissons_ratio')::float, 0), 
	       COALESCE((specific_properties->>'processing_temp_min_c')::float, 0), COALESCE((specific_properties->>'processing_temp_max_c')::float, 0), 
	       COALESCE((specific_properties->>'crystallinity')::float, 0), COALESCE((specific_properties->>'crystal_system')::text, ''), 
	       COALESCE((specific_properties->>'fracture_toughness')::float, 0), COALESCE((specific_properties->>'weibull_modulus')::float, 0), 
	       COALESCE((specific_properties->>'interlaminar_shear_strength')::float, 0), COALESCE((specific_properties->>'fiber_volume_fraction')::float, 0), 
	       COALESCE(source, '')
`

func scanMaterialCandidate(scanFn func(dest ...any) error) (domain.MaterialCandidate, error) {
	var c domain.MaterialCandidate
	err := scanFn(
		&c.ID, &c.Name, &c.Formula, &c.Category, &c.Subcategory,
		&c.Density, &c.GlassTransitionTemp, &c.HeatDeflectionTemp, &c.MeltingPoint, &c.BoilingPoint,
		&c.ThermalConductivity, &c.SpecificHeat, &c.ThermalExpansion, &c.ElectricalResistivity,
		&c.YieldStrength, &c.TensileStrength, &c.YoungsModulus, &c.HardnessVickers, &c.PoissonsRatio,
		&c.ProcessingTempMin, &c.ProcessingTempMax, &c.Crystallinity, &c.CrystalSystem,
		&c.FractureToughness, &c.WeibullModulus, &c.InterlaminarShearStrength, &c.FiberVolumeFraction, &c.Source,
	)
	return c, err
}

func (r *materialRepository) SearchByVector(ctx context.Context, embedding []float32, limit int, categoryFilter string) ([]domain.MaterialCandidate, error) {
	sqlQuery := `
		SELECT id, name, COALESCE(formula, ''), COALESCE(category, ''), COALESCE(subcategory, ''), 
	       COALESCE((specific_properties->>'density')::float, 0), COALESCE((specific_properties->>'glass_transition_temp')::float, 0), 
	       COALESCE((specific_properties->>'heat_deflection_temp')::float, 0), COALESCE((specific_properties->>'melting_point')::float, 0), 
	       COALESCE((specific_properties->>'boiling_point')::float, 0), COALESCE((specific_properties->>'thermal_conductivity')::float, 0), 
	       COALESCE((specific_properties->>'specific_heat')::float, 0), COALESCE((specific_properties->>'thermal_expansion')::float, 0), 
	       COALESCE((specific_properties->>'electrical_resistivity')::float, 0), COALESCE((specific_properties->>'yield_strength')::float, 0), 
	       COALESCE((specific_properties->>'tensile_strength')::float, 0), COALESCE((specific_properties->>'youngs_modulus')::float, 0), 
	       COALESCE((specific_properties->>'hardness_vickers')::float, 0), COALESCE((specific_properties->>'poissons_ratio')::float, 0), 
	       COALESCE((specific_properties->>'processing_temp_min_c')::float, 0), COALESCE((specific_properties->>'processing_temp_max_c')::float, 0), 
	       COALESCE((specific_properties->>'crystallinity')::float, 0), COALESCE((specific_properties->>'crystal_system')::text, ''), 
	       COALESCE((specific_properties->>'fracture_toughness')::float, 0), COALESCE((specific_properties->>'weibull_modulus')::float, 0), 
	       COALESCE((specific_properties->>'interlaminar_shear_strength')::float, 0), COALESCE((specific_properties->>'fiber_volume_fraction')::float, 0), 
	       COALESCE(source, '')
		FROM materials
		WHERE ($3 = '' OR category ILIKE $3)
		ORDER BY embedding <=> $1
		LIMIT $2;
	`
	vectorStr, err := json.Marshal(embedding)
	if err != nil {
		return nil, err
	}

	rows, err := r.pool.Query(ctx, sqlQuery, string(vectorStr), limit, categoryFilter)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.MaterialCandidate
	for rows.Next() {
		candidate, err := scanMaterialCandidate(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, candidate)
	}
	return out, rows.Err()
}

func (r *materialRepository) SearchByIntent(ctx context.Context, intent domain.SearchIntent, query string, limit int) ([]domain.MaterialCandidate, error) {
	clauses := []string{"(name ILIKE $1 OR category ILIKE $1 OR formula ILIKE $1 OR subcategory ILIKE $1)"}
	args := []any{"%" + strings.ReplaceAll(query, " ", "%") + "%"}

	if intent.Domain != "" {
		args = append(args, "%"+intent.Domain+"%")
		clauses = append(clauses, fmt.Sprintf("(category ILIKE $%d OR subcategory ILIKE $%d)", len(args), len(args)))
	}

	if intent.MinYieldStrength != nil {
		args = append(args, *intent.MinYieldStrength)
		clauses = append(clauses, fmt.Sprintf("(specific_properties->>'yield_strength')::float >= $%d", len(args)))
	}

	if intent.MaxDensity != nil {
		args = append(args, *intent.MaxDensity)
		clauses = append(clauses, fmt.Sprintf("(specific_properties->>'density')::float <= $%d", len(args)))
	}

	if intent.MinOperatingTemperature != nil {
		minTemp := *intent.MinOperatingTemperature
		args = append(args, minTemp)
		argIdx1 := len(args)
		args = append(args, minTemp+100) // safety factor
		argIdx2 := len(args)
		
		clauses = append(clauses, fmt.Sprintf(`(
			((specific_properties->>'glass_transition_temp')::float >= $%d) OR 
			((specific_properties->>'glass_transition_temp') IS NULL AND (specific_properties->>'melting_point')::float >= $%d)
		)`, argIdx1, argIdx2))
	}

	args = append(args, limit)
	sqlQuery := fmt.Sprintf(`
		%s
		FROM materials
		WHERE %s
		ORDER BY
			CASE WHEN (specific_properties->>'yield_strength') IS NULL THEN 1 ELSE 0 END,
			(specific_properties->>'yield_strength')::float DESC NULLS LAST,
			name ASC
		LIMIT $%d;
	`, selectAllCols, strings.Join(clauses, " AND "), len(args))

	rows, err := r.pool.Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.MaterialCandidate
	for rows.Next() {
		candidate, err := scanMaterialCandidate(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, candidate)
	}
	return out, rows.Err()
}
