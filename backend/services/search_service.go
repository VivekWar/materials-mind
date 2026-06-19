// Package services orchestrates business logic, caching, and external API integrations.
package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/google/generative-ai-go/genai"
	"github.com/vivekwar/materials-mind/backend/domain"
	"github.com/vivekwar/materials-mind/backend/repositories"
	"github.com/vivekwar/materials-mind/backend/utils"
	"github.com/vivekwar/materials-mind/backend/db" // For redis cache
)

const (
	maxSearchResults    = 5
	maxLLMParseAttempts = 2
	searchCacheTTL      = 30 * time.Minute
)

// SearchService defines the interface for core retrieval and generation logic.
type SearchService interface {
	ProcessSearch(ctx context.Context, query string, industryDomain string) (*domain.StructuredRecommendation, error)
	ProcessSearchStream(ctx context.Context, query string, industryDomain string) (<-chan domain.StreamEvent, error)
	ProcessFollowup(ctx context.Context, req domain.FollowUpChatRequest) (*domain.FollowUpChatResponse, error)
	GenerateTitle(ctx context.Context, query string) (string, error)
}

type searchService struct {
	repo repositories.MaterialRepository
}

func NewSearchService(repo repositories.MaterialRepository) SearchService {
	return &searchService{repo: repo}
}

func (s *searchService) ProcessSearch(ctx context.Context, query string, industryDomain string) (*domain.StructuredRecommendation, error) {
	normalizedQuery := s.normalizeSearchQuery(query)

	// Check cache IMMEDIATELY based on the query to save all Gemini API calls!
	cacheKey := s.cacheKeyFor("search:v2", normalizedQuery+":"+industryDomain)
	if cached, ok := s.getCachedStructuredRecommendation(ctx, cacheKey); ok {
		return cached, nil
	}

	type intentResult struct {
		intent domain.SearchIntent
		err    error
	}
	intentCh := make(chan intentResult, 1)

	// Run Intent Extraction concurrently
	go func() {
		intent, err := s.extractSearchIntent(ctx, normalizedQuery)
		intentCh <- intentResult{intent, err}
	}()

	type resultOrErr struct {
		candidates []domain.MaterialCandidate
		err        error
	}

	vecCh := make(chan resultOrErr, 1)
	kwCh := make(chan resultOrErr, 1)

	// Vector path (starts IMMEDIATELY, fully parallel with intent extraction)
	go func() {
		var v []float32
		var embErr error
		err := utils.Backoff(ctx, 3, 500*time.Millisecond, 5*time.Second, func() error {
			v, embErr = GetEmbedding(ctx, normalizedQuery)
			return embErr
		})
		if err != nil {
			vecCh <- resultOrErr{nil, fmt.Errorf("embedding failed: %w", err)}
			return
		}

		out, qErr := s.repo.SearchByVector(ctx, v, maxSearchResults, "")
		vecCh <- resultOrErr{out, qErr}
	}()

	// Wait for intent to start keyword search
	iRes := <-intentCh
	intent := iRes.intent
	if iRes.err != nil {
		log.Printf("intent extraction failed: %v", iRes.err)
	}

	// Keyword path
	go func() {
		out, qErr := s.repo.SearchByIntent(ctx, intent, normalizedQuery, maxSearchResults)
		kwCh <- resultOrErr{out, qErr}
	}()

	vecRes := <-vecCh
	kwRes := <-kwCh

	if vecRes.err != nil {
		log.Printf("vector path error: %v", vecRes.err)
	}
	if kwRes.err != nil {
		log.Printf("keyword path error: %v", kwRes.err)
	}

	merged := make([]domain.MaterialCandidate, 0, maxSearchResults)
	seen := make(map[int64]struct{})
	for _, m := range vecRes.candidates {
		if _, ok := seen[m.ID]; ok {
			continue
		}
		merged = append(merged, m)
		seen[m.ID] = struct{}{}
		if len(merged) >= maxSearchResults {
			break
		}
	}
	if len(merged) < maxSearchResults {
		for _, m := range kwRes.candidates {
			if _, ok := seen[m.ID]; ok {
				continue
			}
			merged = append(merged, m)
			seen[m.ID] = struct{}{}
			if len(merged) >= maxSearchResults {
				break
			}
		}
	}

	if len(merged) == 0 {
		return &domain.StructuredRecommendation{
			RecommendedMaterial: "None Found",
			Confidence:          "Low",
			ConfidenceScore:     0.0,
			WhyItMatches:        []string{"No materials in the database meet your strict physical or thermal constraints."},
			TradeOffs:           []string{"Consider relaxing your temperature or strength requirements."},
			Report:              "I could not find matching materials for this query. Please add more constraints like strength, weight, temperature, or budget.",
		}, nil
	}

	// Apply Domain-Awareness Multiplier
	if industryDomain != "" {
		domainLower := strings.ToLower(industryDomain)
		for i := range merged {
			cat := strings.ToLower(merged[i].Category)
			if strings.Contains(cat, domainLower) || strings.Contains(domainLower, cat) {
				// We don't have a score field on Candidate, so we just rely on passing the domain to the LLM
			}
		}
	}

	model := GetGenerativeModel()
	recommendation, recErr := s.generateStructuredRecommendation(ctx, model, normalizedQuery, industryDomain, merged)
	if recErr != nil {
		return nil, recErr
	}

	recommendation.Candidates = merged

	s.cacheStructuredRecommendation(ctx, cacheKey, recommendation)
	return recommendation, nil
}

func (s *searchService) normalizeSearchQuery(query string) string {
	return strings.Join(strings.Fields(query), " ")
}

func (s *searchService) extractSearchIntent(ctx context.Context, query string) (domain.SearchIntent, error) {
	prompt := fmt.Sprintf(`You are a data extraction assistant for a materials engineering database.
Extract physical requirements and business constraints from the user query into a JSON object.

Output strictly this JSON schema (do not use markdown formatting, just raw JSON):
{
  "domain": "string (e.g. Aerospace, Automotive)",
  "min_yield_strength": number (MPa),
  "min_operating_temperature": number (Celsius),
  "max_density": number (g/cm3),
  "budget_constraint": "string (e.g. low, medium, high)"
}
Only populate fields that are explicitly or implicitly requested. For example, if 'lightweight' is requested, you may set a reasonable max_density (like 3.5). If 130C is requested, set min_operating_temperature.

User Query: %s`, query)

	if GeminiClient == nil {
		return domain.SearchIntent{}, errors.New("gemini client not initialized")
	}
	model := GetGenerativeModel()
	model.ResponseMIMEType = "application/json"
	
	var resp *genai.GenerateContentResponse
	var err error
	utils.Backoff(ctx, 5, 5*time.Second, 60*time.Second, func() error {
		resp, err = model.GenerateContent(ctx, genai.Text(prompt))
		return err
	})

	if err != nil {
		return domain.SearchIntent{}, err
	}

	text, extErr := s.extractModelText(resp)
	if extErr != nil {
		return domain.SearchIntent{}, extErr
	}

	var intent domain.SearchIntent
	clean := strings.TrimSpace(text)
	clean = strings.TrimPrefix(clean, "```json")
	clean = strings.TrimPrefix(clean, "```")
	clean = strings.TrimSuffix(clean, "```")
	clean = strings.TrimSpace(clean)
	
	if err := json.Unmarshal([]byte(clean), &intent); err != nil {
		return domain.SearchIntent{}, fmt.Errorf("failed to parse intent json: %w", err)
	}

	return intent, nil
}

func (s *searchService) cacheKeyFor(prefix, value string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(value))))
	return prefix + ":" + hex.EncodeToString(sum[:])
}

func (s *searchService) getCachedStructuredRecommendation(ctx context.Context, key string) (*domain.StructuredRecommendation, bool) {
	if db.RedisClient == nil {
		return nil, false
	}
	cached, err := db.RedisClient.Get(ctx, key).Result()
	if err != nil || cached == "" {
		return nil, false
	}
	var out domain.StructuredRecommendation
	if err := json.Unmarshal([]byte(cached), &out); err != nil {
		return nil, false
	}
	return &out, true
}

func (s *searchService) cacheStructuredRecommendation(ctx context.Context, key string, rec *domain.StructuredRecommendation) {
	if db.RedisClient == nil || rec == nil {
		return
	}
	value, err := json.Marshal(rec)
	if err != nil {
		return
	}
	if err := db.RedisClient.Set(ctx, key, string(value), searchCacheTTL).Err(); err != nil {
		log.Printf("search cache set failed: %v", err)
	}
}

func (s *searchService) generateStructuredRecommendation(
	ctx context.Context,
	model *genai.GenerativeModel,
	query string,
	industryDomain string,
	candidates []domain.MaterialCandidate,
) (*domain.StructuredRecommendation, error) {
	model.ResponseMIMEType = "application/json"
	prompt := s.buildStructuredRecommendationPrompt(query, industryDomain, candidates)
	for attempt := 1; attempt <= maxLLMParseAttempts; attempt++ {
		var response *genai.GenerateContentResponse
		var llmErr error

		err := utils.Backoff(ctx, 5, 5*time.Second, 60*time.Second, func() error {
			response, llmErr = model.GenerateContent(ctx, genai.Text(prompt))
			return llmErr
		})

		if err != nil {
			return nil, fmt.Errorf("model generation failed: %w", err)
		}

		modelText, extractErr := s.extractModelText(response)
		if extractErr != nil {
			if attempt == maxLLMParseAttempts {
				return nil, fmt.Errorf("response extraction failed: %w", extractErr)
			}
			prompt = s.buildStructuredRepairPrompt("")
			continue
		}

		recommendation, parseErr := s.parseStructuredRecommendation(modelText)
		if parseErr == nil {
			if validateErr := s.validateRecommendation(recommendation, candidates); validateErr == nil {
				return recommendation, nil
			}
		}

		if attempt == maxLLMParseAttempts {
			if parseErr != nil {
				return nil, fmt.Errorf("structured parse failed: %w", parseErr)
			}
			return nil, errors.New("structured response failed validation")
		}

		prompt = s.buildStructuredRepairPrompt(modelText)
	}

	return nil, errors.New("structured generation exceeded retries")
}

func (s *searchService) buildStructuredRecommendationPrompt(query string, industryDomain string, candidates []domain.MaterialCandidate) string {
	var materialContext strings.Builder
	for _, candidate := range candidates {
		if candidate.YieldStrength > 0 {
			materialContext.WriteString(
				fmt.Sprintf("- [ID:%d] %s | Category: %s | Yield Strength: %.2f MPa\n",
					candidate.ID, candidate.Name, candidate.Category, candidate.YieldStrength),
			)
		} else {
			materialContext.WriteString(
				fmt.Sprintf("- [ID:%d] %s | Category: %s\n",
					candidate.ID, candidate.Name, candidate.Category),
			)
		}
	}

	domainContext := ""
	if industryDomain != "" {
		domainContext = fmt.Sprintf("Industry Domain Context: %s\n", industryDomain)
	}

	return fmt.Sprintf(`You are an expert, friendly, and highly knowledgeable Principal Materials Scientist. You are collaborating with a fellow engineer to help them select the perfect material for their project. Your tone should be conversational, helpful, and natural—not robotic.

When evaluating materials or answering questions, apply these rules:
1. BE CONVERSATIONAL & HUMAN: Speak naturally. Do NOT use stiff, robotic bullet points like "Pros:", "Cons:", or "Tradeoffs:". Weave the tradeoffs naturally into your paragraphs.
2. NO STUBBORNNESS: If a valid physical property conflict exists, gracefully pivot your recommendation. 
3. QUANTITATIVE MATH OVER QUALITATIVE TEXT: Calculate density/cost differences if financial penalties exist, but explain them naturally like a colleague would.
4. AGNOSTIC TO BRANDS/LABELS: Evaluate strictly by crystal structures and engineering properties. Remember FCC metals excel in cryogenics, while BCC and HCP metals are highly prone to low-temperature embrittlement.
5. ACKNOWLEDGE LIMITATIONS: If retrieved data lacks exact alloy grades, point this out as a helpful warning.

User query: %q

%s
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
  "report": "A natural, multi-paragraph conversation from a senior engineer to a colleague."
}

Formatting & System Rules:
- "sources" must only include IDs from retrieved materials.
- "confidence_score" must be a precise float between 0.000 and 1.000 (e.g., 0.914).
- "report" MUST NOT contain raw data dumps (like listing melting points or density). The user already has a Data Sheet UI for raw numbers! The report is for your engineering rationale, the tradeoffs written in prose, and helpful contextual advice.
- Do not include extra keys.`, query, domainContext, materialContext.String())
}

func (s *searchService) buildStructuredRepairPrompt(rawOutput string) string {
	return fmt.Sprintf(`Convert the following output into valid JSON only (no markdown, no code fences) with this exact schema:
{
  "recommended_material": "string",
  "why_it_matches": ["string", "string"],
  "trade_offs": ["string"],
  "confidence": "High|Medium|Low",
  "confidence_score": 0.0,
  "sources": [12, 45],
  "report": "detailed, comprehensive multi-paragraph engineer explanation"
}

Keep all factual content intact and do not invent new source IDs.
Text to repair:
%s`, rawOutput)
}

func (s *searchService) extractModelText(resp *genai.GenerateContentResponse) (string, error) {
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

func (s *searchService) parseStructuredRecommendation(raw string) (*domain.StructuredRecommendation, error) {
	clean := strings.TrimSpace(raw)
	clean = strings.TrimPrefix(clean, "```json")
	clean = strings.TrimPrefix(clean, "```")
	clean = strings.TrimSuffix(clean, "```")
	clean = strings.TrimSpace(clean)

	var out domain.StructuredRecommendation
	if err := json.Unmarshal([]byte(clean), &out); err != nil {
		return nil, fmt.Errorf("unmarshal structured response: %w", err)
	}

	return &out, nil
}

func (s *searchService) validateRecommendation(rec *domain.StructuredRecommendation, candidates []domain.MaterialCandidate) error {
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

func (s *searchService) ProcessFollowup(ctx context.Context, req domain.FollowUpChatRequest) (*domain.FollowUpChatResponse, error) {
	if GeminiClient == nil {
		return nil, errors.New("gemini client is not initialized")
	}

	prompt := s.buildFollowUpPrompt(req)
	cacheKey := s.cacheKeyFor("chat-followup:v1", prompt)
	if cached, ok := s.getCachedFollowup(ctx, cacheKey); ok {
		return cached, nil
	}

	model := GetGenerativeModel()

	var resp *genai.GenerateContentResponse
	var err error

	backoffErr := utils.Backoff(ctx, 3, 1*time.Second, 10*time.Second, func() error {
		resp, err = model.GenerateContent(ctx, genai.Text(prompt))
		return err
	})

	if backoffErr != nil {
		return nil, fmt.Errorf("follow-up generation failed: %w", backoffErr)
	}

	text, err := s.extractModelText(resp)
	if err != nil {
		return nil, errors.New("empty follow-up response")
	}

	out := domain.FollowUpChatResponse{
		Reply:      strings.TrimSpace(text),
		TokensUsed: 0,
	}
	s.cacheFollowup(ctx, cacheKey, out)
	return &out, nil
}

func (s *searchService) buildFollowUpPrompt(req domain.FollowUpChatRequest) string {
	var historyBuilder strings.Builder
	for _, turn := range req.History {
		role := strings.ToLower(strings.TrimSpace(turn.Role))
		if role != "user" && role != "assistant" {
			continue
		}
		content := strings.TrimSpace(turn.Content)
		if content == "" {
			continue
		}
		historyBuilder.WriteString(fmt.Sprintf("%s: %s\n", strings.Title(role), content))
	}

	var topRecommendations strings.Builder
	for _, item := range req.TopRecommendations {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		topRecommendations.WriteString("- ")
		topRecommendations.WriteString(item)
		topRecommendations.WriteString("\n")
	}

	return fmt.Sprintf(`You are an expert, friendly, and highly conversational Principal Materials Scientist.

When users challenge your recommendations or ask follow-up questions, apply these rules:
1. BE CONVERSATIONAL: Speak naturally and warmly like an expert colleague. Do NOT use stiff formatting or bullet points for "Pros/Cons". Weave the explanation naturally into prose.
2. NO STUBBORNNESS: If a user points out a valid physical property conflict, gracefully pivot your recommendation. Do not defend a suboptimal material.
3. QUANTITATIVE MATH OVER QUALITATIVE TEXT: If the user provides a financial penalty constraint, explicitly calculate or weigh the density differences.
4. AGNOSTIC TO BRANDS/LABELS: Evaluate materials strictly by their crystal structures and engineering properties.
5. ACKNOWLEDGE LIMITATIONS: Point out missing data immediately rather than assuming suitability for extreme environments.

Conversation history:
%s

My initial analysis/recommendation:
%s

Top recommendations:
%s

User follow-up:
%s

System Rules:
- Answer the user's follow-up question directly in a conversational, first-person tone.
- Do NOT refer to "the report" in the third person. Take ownership of the previous analysis.
- Stay grounded in your initial analysis.
- Do not invent new material properties.
- Provide a helpful, comprehensive, and friendly explanation. Do NOT dump raw data lists.`, historyBuilder.String(), strings.TrimSpace(req.InitialReport), topRecommendations.String(), strings.TrimSpace(req.Message))
}

func (s *searchService) getCachedFollowup(ctx context.Context, key string) (*domain.FollowUpChatResponse, bool) {
	if db.RedisClient == nil {
		return nil, false
	}
	cached, err := db.RedisClient.Get(ctx, key).Result()
	if err != nil || cached == "" {
		return nil, false
	}
	var out domain.FollowUpChatResponse
	if err := json.Unmarshal([]byte(cached), &out); err != nil {
		return nil, false
	}
	return &out, true
}

func (s *searchService) cacheFollowup(ctx context.Context, key string, value domain.FollowUpChatResponse) {
	if db.RedisClient == nil {
		return
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return
	}
	if err := db.RedisClient.Set(ctx, key, string(payload), searchCacheTTL).Err(); err != nil {
		log.Printf("follow-up cache set failed: %v", err)
	}
}

func (s *searchService) GenerateTitle(ctx context.Context, query string) (string, error) {
	client := GeminiClient
	if client == nil {
		return "New chat", errors.New("gemini client not initialized")
	}

	model := client.GenerativeModel("gemini-3.5-flash")
	model.SetTemperature(0.7)

	prompt := fmt.Sprintf(`Generate a very concise, 2-to-4 word topic title for a chat session based on this user's first message. 
Do not include quotes or punctuation. Just the title.

User Message: "%s"`, query)

	fallbackTitle := func(q string) string {
		words := strings.Fields(q)
		if len(words) > 4 {
			return strings.Join(words[:4], " ") + "..."
		}
		if len(words) > 0 {
			return q
		}
		return "New chat"
	}

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return fallbackTitle(query), nil
	}

	if len(resp.Candidates) > 0 && len(resp.Candidates[0].Content.Parts) > 0 {
		part := resp.Candidates[0].Content.Parts[0]
		if textPart, ok := part.(genai.Text); ok {
			title := strings.TrimSpace(string(textPart))
			title = strings.Trim(title, "\"'")
			if title != "" && !strings.EqualFold(title, "new chat") && !strings.EqualFold(title, "new session") {
				return title, nil
			}
		}
	}

	return fallbackTitle(query), nil
}

func (s *searchService) ProcessSearchStream(ctx context.Context, query string, industryDomain string) (<-chan domain.StreamEvent, error) {
	out := make(chan domain.StreamEvent, 100)
	
	go func() {
		defer close(out)
		
		recommendation, err := s.ProcessSearch(ctx, query, industryDomain)
		if err != nil {
			out <- domain.StreamEvent{Type: "error", Data: "An error occurred while processing your search."}
			return
		}

		structuredJSON, err := json.Marshal(recommendation)
		if err != nil {
			out <- domain.StreamEvent{Type: "error", Data: "Failed to format generated response"}
			return
		}

		out <- domain.StreamEvent{Type: "structured_result", Data: string(structuredJSON)}
		
		out <- domain.StreamEvent{Type: "message", Data: recommendation.Report}
		out <- domain.StreamEvent{Type: "done", Data: "true"}
	}()
	
	return out, nil
}
