package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/google/generative-ai-go/genai"
	"github.com/vivekwar/materialmind/db"
	"github.com/vivekwar/materialmind/services"
)

const (
	minSearchQueryRunes = 3
	maxSearchResults    = 5
	maxLLMParseAttempts = 2
)

type SearchRequest struct {
	Query string `json:"query" binding:"required,min=3,max=500"`
}

type materialCandidate struct {
	ID            int64
	Name          string
	Category      string
	YieldStrength float64
}

type structuredRecommendation struct {
	RecommendedMaterial string   `json:"recommended_material"`
	WhyItMatches        []string `json:"why_it_matches"`
	TradeOffs           []string `json:"trade_offs"`
	Confidence          string   `json:"confidence"`
	ConfidenceScore     float64  `json:"confidence_score"`
	Sources             []int64  `json:"sources"`
	Report              string   `json:"report"`
}

func HybridSearch(c *gin.Context) {
	var req SearchRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON body"})
		return
	}

	normalizedQuery := normalizeSearchQuery(req.Query)
	if utf8.RuneCountInString(normalizedQuery) < minSearchQueryRunes {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query is too short"})
		return
	}

	// SSE is used so the client can render model output incrementally.
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Status(http.StatusOK)

	vector, err := services.GetEmbedding(c.Request.Context(), normalizedQuery)
	if err != nil {
		log.Printf("embedding generation failed: %v", err)
		c.SSEvent("error", "Failed to process query vector")
		c.Writer.Flush()
		return
	}

	// Vector index is assumed to be pre-populated and production-ready.
	sqlQuery := `
		SELECT m.id, m.name, m.category, m.yield_strength
		FROM materials m
		JOIN material_embeddings me ON m.id = me.material_id
		ORDER BY me.embedding <=> $1
		LIMIT $2;
	`

	vectorStr, err := json.Marshal(vector)
	if err != nil {
		log.Printf("vector marshal failed: %v", err)
		c.SSEvent("error", "Failed to prepare vector query")
		c.Writer.Flush()
		return
	}

	rows, err := db.Pool.Query(c.Request.Context(), sqlQuery, string(vectorStr), maxSearchResults)
	if err != nil {
		log.Printf("database search failed: %v", err)
		c.SSEvent("error", "Database search failed")
		c.Writer.Flush()
		return
	}
	defer rows.Close()

	candidates := make([]materialCandidate, 0, maxSearchResults)
	for rows.Next() {
		var candidate materialCandidate
		if scanErr := rows.Scan(&candidate.ID, &candidate.Name, &candidate.Category, &candidate.YieldStrength); scanErr != nil {
			log.Printf("row scan failed: %v", scanErr)
			c.SSEvent("error", "Failed to process search results")
			c.Writer.Flush()
			return
		}
		candidates = append(candidates, candidate)
	}

	if rowsErr := rows.Err(); rowsErr != nil {
		log.Printf("row iteration failed: %v", rowsErr)
		c.SSEvent("error", "Failed while reading search results")
		c.Writer.Flush()
		return
	}

	if len(candidates) == 0 {
		c.SSEvent("message", "I could not find matching materials for this query. Please add more constraints like strength, weight, temperature, or budget.")
		c.SSEvent("done", "true")
		c.Writer.Flush()
		return
	}

	model := services.GeminiClient.GenerativeModel(services.GenerativeModelName)
	recommendation, recErr := generateStructuredRecommendation(c, model, normalizedQuery, candidates)
	if recErr != nil {
		log.Printf("structured generation failed: %v", recErr)
		c.SSEvent("error", "Generated response failed validation")
		c.Writer.Flush()
		return
	}

	structuredJSON, marshalErr := json.Marshal(recommendation)
	if marshalErr != nil {
		log.Printf("structured marshal failed: %v", marshalErr)
		c.SSEvent("error", "Failed to format generated response")
		c.Writer.Flush()
		return
	}

	// Keep existing clients compatible while exposing structured output.
	c.SSEvent("structured_result", string(structuredJSON))
	c.SSEvent("message", recommendation.Report)
	c.SSEvent("done", "true")
	c.Writer.Flush()
}

func normalizeSearchQuery(query string) string {
	return strings.Join(strings.Fields(query), " ")
}

func buildStructuredRecommendationPrompt(query string, candidates []materialCandidate) string {
	var materialContext strings.Builder
	for _, candidate := range candidates {
		materialContext.WriteString(
			fmt.Sprintf("- [ID:%d] %s | Category: %s | Yield Strength: %.2f MPa\n",
				candidate.ID, candidate.Name, candidate.Category, candidate.YieldStrength),
		)
	}

	return fmt.Sprintf(`You are a senior materials engineer.
User query: %q

Only use the retrieved materials below. Do not invent properties.
Retrieved materials:
%s

Return only valid JSON (no markdown, no code fences) with this exact shape:
{
  "recommended_material": "string",
  "why_it_matches": ["string", "string"],
  "trade_offs": ["string"],
  "confidence": "High|Medium|Low",
  "confidence_score": 0.0,
  "sources": [12, 45],
  "report": "short multi-line engineer explanation"
}

Rules:
- "sources" must only include IDs from retrieved materials.
- "confidence_score" must be a number from 0.0 to 1.0.
- "report" should be concise and reference the chosen material and key trade-offs.
- Do not include extra keys.`, query, materialContext.String())
}

func buildStructuredRepairPrompt(rawOutput string) string {
	return fmt.Sprintf(`Convert the following output into valid JSON only (no markdown, no code fences) with this exact schema:
{
  "recommended_material": "string",
  "why_it_matches": ["string", "string"],
  "trade_offs": ["string"],
  "confidence": "High|Medium|Low",
  "confidence_score": 0.0,
  "sources": [12, 45],
  "report": "short multi-line engineer explanation"
}

Keep all factual content intact and do not invent new source IDs.
Text to repair:
%s`, rawOutput)
}

func generateStructuredRecommendation(
	c *gin.Context,
	model *genai.GenerativeModel,
	query string,
	candidates []materialCandidate,
) (*structuredRecommendation, error) {
	prompt := buildStructuredRecommendationPrompt(query, candidates)
	for attempt := 1; attempt <= maxLLMParseAttempts; attempt++ {
		response, llmErr := model.GenerateContent(c.Request.Context(), genai.Text(prompt))
		if llmErr != nil {
			return nil, fmt.Errorf("model generation failed: %w", llmErr)
		}

		modelText, extractErr := extractModelText(response)
		if extractErr != nil {
			if attempt == maxLLMParseAttempts {
				return nil, fmt.Errorf("response extraction failed: %w", extractErr)
			}
			prompt = buildStructuredRepairPrompt("")
			continue
		}

		recommendation, parseErr := parseStructuredRecommendation(modelText)
		if parseErr == nil {
			if validateErr := validateRecommendation(recommendation, candidates); validateErr == nil {
				return recommendation, nil
			}
		}

		if attempt == maxLLMParseAttempts {
			if parseErr != nil {
				return nil, fmt.Errorf("structured parse failed: %w", parseErr)
			}
			return nil, errors.New("structured response failed validation")
		}

		prompt = buildStructuredRepairPrompt(modelText)
	}

	return nil, errors.New("structured generation exceeded retries")
}

func extractModelText(resp *genai.GenerateContentResponse) (string, error) {
	if resp == nil {
		return "", errors.New("nil model response")
	}

	var builder strings.Builder
	for _, candidate := range resp.Candidates {
		if candidate == nil || candidate.Content == nil {
			continue
		}
		for _, part := range candidate.Content.Parts {
			builder.WriteString(fmt.Sprint(part))
		}
	}

	text := strings.TrimSpace(builder.String())
	if text == "" {
		return "", errors.New("no text in model response")
	}

	return text, nil
}

func parseStructuredRecommendation(raw string) (*structuredRecommendation, error) {
	clean := strings.TrimSpace(raw)
	clean = strings.TrimPrefix(clean, "```json")
	clean = strings.TrimPrefix(clean, "```")
	clean = strings.TrimSuffix(clean, "```")
	clean = strings.TrimSpace(clean)

	var out structuredRecommendation
	if err := json.Unmarshal([]byte(clean), &out); err != nil {
		return nil, fmt.Errorf("unmarshal structured response: %w", err)
	}

	return &out, nil
}

func validateRecommendation(rec *structuredRecommendation, candidates []materialCandidate) error {
	if rec == nil {
		return errors.New("nil recommendation")
	}
	if strings.TrimSpace(rec.RecommendedMaterial) == "" {
		return errors.New("recommended_material is required")
	}
	if len(rec.WhyItMatches) == 0 {
		return errors.New("why_it_matches is required")
	}
	if len(rec.TradeOffs) == 0 {
		return errors.New("trade_offs is required")
	}
	if strings.TrimSpace(rec.Report) == "" {
		return errors.New("report is required")
	}

	confidence := strings.ToLower(strings.TrimSpace(rec.Confidence))
	if confidence != "high" && confidence != "medium" && confidence != "low" {
		return errors.New("confidence must be High, Medium, or Low")
	}
	if rec.ConfidenceScore < 0 || rec.ConfidenceScore > 1 {
		return errors.New("confidence_score must be between 0.0 and 1.0")
	}

	if len(rec.Sources) == 0 {
		return errors.New("at least one source id is required")
	}

	allowed := make(map[int64]struct{}, len(candidates))
	for _, c := range candidates {
		allowed[c.ID] = struct{}{}
	}

	uniqueSources := make(map[int64]struct{}, len(rec.Sources))
	for _, srcID := range rec.Sources {
		if _, ok := allowed[srcID]; !ok {
			return fmt.Errorf("source id %d not in retrieved candidates", srcID)
		}
		uniqueSources[srcID] = struct{}{}
	}

	normalized := make([]int64, 0, len(uniqueSources))
	for srcID := range uniqueSources {
		normalized = append(normalized, srcID)
	}
	sort.Slice(normalized, func(i, j int) bool { return normalized[i] < normalized[j] })
	rec.Sources = normalized

	switch confidence {
	case "high":
		rec.Confidence = "High"
	case "medium":
		rec.Confidence = "Medium"
	case "low":
		rec.Confidence = "Low"
	}

	return nil
}
