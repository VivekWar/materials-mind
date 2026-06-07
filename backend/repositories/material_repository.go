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

	if intent.Category != "" {
		args = append(args, intent.Category)
		clauses = append(clauses, fmt.Sprintf("category ILIKE $%d", len(args)))
	}

	allowedFields := map[string]struct{}{
		"density": {}, "yield_strength": {}, "youngs_modulus": {}, "thermal_conductivity": {}, "melting_point": {},
	}
	for _, filter := range intent.Filters {
		if _, ok := allowedFields[filter.Field]; !ok {
			continue
		}
		if filter.Minimum != nil {
			args = append(args, *filter.Minimum)
			clauses = append(clauses, fmt.Sprintf("%s >= $%d", filter.Field, len(args)))
		}
		if filter.Maximum != nil {
			args = append(args, *filter.Maximum)
			clauses = append(clauses, fmt.Sprintf("%s <= $%d", filter.Field, len(args)))
		}
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
