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
	       COALESCE(density, 0), COALESCE(glass_transition_temp, 0), COALESCE(heat_deflection_temp, 0), COALESCE(melting_point, 0), COALESCE(boiling_point, 0), 
	       COALESCE(thermal_conductivity, 0), COALESCE(specific_heat, 0), COALESCE(thermal_expansion, 0), COALESCE(electrical_resistivity, 0), 
	       COALESCE(yield_strength, 0), COALESCE(tensile_strength, 0), COALESCE(youngs_modulus, 0), COALESCE(hardness_vickers, 0), COALESCE(poissons_ratio, 0), 
	       COALESCE(processing_temp_min_c, 0), COALESCE(processing_temp_max_c, 0), COALESCE(crystallinity, 0), COALESCE(crystal_system, ''), 
	       COALESCE(fracture_toughness, 0), COALESCE(weibull_modulus, 0), COALESCE(interlaminar_shear_strength, 0), COALESCE(fiber_volume_fraction, 0), COALESCE(source, '')
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
		SELECT m.id, m.name, COALESCE(m.formula, ''), COALESCE(m.category, ''), COALESCE(m.subcategory, ''), 
		       COALESCE(m.density, 0), COALESCE(m.glass_transition_temp, 0), COALESCE(m.heat_deflection_temp, 0), COALESCE(m.melting_point, 0), COALESCE(m.boiling_point, 0), 
		       COALESCE(m.thermal_conductivity, 0), COALESCE(m.specific_heat, 0), COALESCE(m.thermal_expansion, 0), COALESCE(m.electrical_resistivity, 0), 
		       COALESCE(m.yield_strength, 0), COALESCE(m.tensile_strength, 0), COALESCE(m.youngs_modulus, 0), COALESCE(m.hardness_vickers, 0), COALESCE(m.poissons_ratio, 0), 
		       COALESCE(m.processing_temp_min_c, 0), COALESCE(m.processing_temp_max_c, 0), COALESCE(m.crystallinity, 0), COALESCE(m.crystal_system, ''), 
		       COALESCE(m.fracture_toughness, 0), COALESCE(m.weibull_modulus, 0), COALESCE(m.interlaminar_shear_strength, 0), COALESCE(m.fiber_volume_fraction, 0), COALESCE(m.source, '')
		FROM materials m
		JOIN material_embeddings me ON m.id = me.material_id
		WHERE ($3 = '' OR m.category ILIKE $3)
		ORDER BY me.embedding <=> $1
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
		clauses = append(clauses, fmt.Sprintf("yield_strength >= $%d", len(args)))
	}

	if intent.MaxDensity != nil {
		args = append(args, *intent.MaxDensity)
		clauses = append(clauses, fmt.Sprintf("density <= $%d", len(args)))
	}

	if intent.MinOperatingTemperature != nil {
		minTemp := *intent.MinOperatingTemperature
		args = append(args, minTemp)
		argIdx1 := len(args)
		args = append(args, minTemp+100) // safety factor
		argIdx2 := len(args)
		
		clauses = append(clauses, fmt.Sprintf(`(
			(glass_transition_temp >= $%d) OR 
			(glass_transition_temp IS NULL AND melting_point >= $%d)
		)`, argIdx1, argIdx2))
	}

	args = append(args, limit)
	sqlQuery := fmt.Sprintf(`
		%s
		FROM materials
		WHERE %s
		ORDER BY
			CASE WHEN yield_strength IS NULL THEN 1 ELSE 0 END,
			yield_strength DESC NULLS LAST,
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
