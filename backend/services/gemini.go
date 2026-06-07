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
	GenerativeModelName   = "gemini-2.5-flash"
	maxEmbeddingRetries   = 3
	maxEmbeddingInputRune = 2000
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
