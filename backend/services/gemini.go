package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

var GeminiClient *genai.Client

const (
	EmbeddingModelName    = "gemini-embedding-001"
	GenerativeModelName   = "gemini-3.1-pro-preview"
	maxEmbeddingRetries   = 3
	maxEmbeddingInputRune = 5000
)

func InitGemini() {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatal("Critical: GEMINI_API_KEY environment variable is missing")
	}

	client, err := genai.NewClient(context.Background(), option.WithAPIKey(apiKey))
	if err != nil {
		log.Fatalf("Critical: Failed to initialize Gemini: %v", err)
	}

	GeminiClient = client
	log.Println("✅ Gemini AI Client Initialized")
}

func GetEmbedding(ctx context.Context, text string) ([]float32, error) {
	if GeminiClient == nil {
		return nil, errors.New("gemini client is not initialized")
	}

	cleanText := normalizeEmbeddingInput(text)
	if cleanText == "" {
		return nil, errors.New("embedding input cannot be empty")
	}

	em := GeminiClient.EmbeddingModel(EmbeddingModelName)
	var lastErr error

	for attempt := 1; attempt <= maxEmbeddingRetries; attempt++ {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		res, err := em.EmbedContent(ctx, genai.Text(cleanText))
		if err == nil {
			if res == nil || res.Embedding == nil || len(res.Embedding.Values) == 0 {
				return nil, errors.New("embedding model returned empty vector")
			}
			
			values := res.Embedding.Values
			// Truncate to 768 dimensions to match the PostgreSQL schema and python seed script
			if len(values) > 768 {
				values = values[:768]
			}
			return values, nil
		}

		lastErr = err
		if attempt == maxEmbeddingRetries {
			break
		}

		backoff := time.Duration(200*(1<<(attempt-1))) * time.Millisecond
		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}

	return nil, fmt.Errorf("embedding generation failed after %d attempts: %w", maxEmbeddingRetries, lastErr)
}

func normalizeEmbeddingInput(text string) string {
	normalized := strings.Join(strings.Fields(text), " ")
	if utf8.RuneCountInString(normalized) <= maxEmbeddingInputRune {
		return normalized
	}

	runes := []rune(normalized)
	return string(runes[:maxEmbeddingInputRune])
}

func GetGenerativeModel() *genai.GenerativeModel {
	if GeminiClient == nil {
		return nil
	}
	model := GeminiClient.GenerativeModel(GenerativeModelName)
	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{
			genai.Text(`CRITICAL Directives for Preventing Hallucinations:

1. YOU CANNOT ALTER REALITY: If a user requests a "lightweight" material, but the best database match has a density > 5.0 g/cm³, you MUST explicitly state: "This material fails the lightweight requirement." Do NOT claim a heavy metal has "relatively low density."

2. CORROSION & GALVANIC STRICTNESS: Never assume a material is corrosion-resistant in seawater or aggressive chemicals unless the database explicitly states it. If bolting two metals together (e.g., Fasteners to a Frame), you MUST evaluate Galvanic Corrosion. If the database does not contain galvanic potential data, you MUST warn the user: "Galvanic compatibility with [Base Metal] must be verified by a metallurgist."

3. DO NOT BE A 'YES MAN': If the vector database returns materials that do not perfectly fit the user's prompt, do not invent properties to make them fit. Present the best available option, but ruthlessly list exactly which constraints it fails in the "Trade-offs" section.`),
		},
	}
	return model
}
