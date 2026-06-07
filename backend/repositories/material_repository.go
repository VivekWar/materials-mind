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

func (r *materialRepository) SearchByVector(ctx context.Context, embedding []float32, limit int, categoryFilter string) ([]domain.MaterialCandidate, error) {
	sqlQuery := `
		SELECT m.id, m.name, COALESCE(m.category, ''), COALESCE(m.yield_strength, 0)
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
		var candidate domain.MaterialCandidate
		if err := rows.Scan(&candidate.ID, &candidate.Name, &candidate.Category, &candidate.YieldStrength); err != nil {
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
		SELECT id, name, COALESCE(category, ''), COALESCE(yield_strength, 0)
		FROM materials
		WHERE %s
		ORDER BY
			CASE WHEN yield_strength IS NULL THEN 1 ELSE 0 END,
			yield_strength DESC NULLS LAST,
			name ASC
		LIMIT $%d;
	`, strings.Join(clauses, " AND "), len(args))

	rows, err := r.pool.Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.MaterialCandidate
	for rows.Next() {
		var candidate domain.MaterialCandidate
		if err := rows.Scan(&candidate.ID, &candidate.Name, &candidate.Category, &candidate.YieldStrength); err != nil {
			return nil, err
		}
		out = append(out, candidate)
	}
	return out, rows.Err()
}
