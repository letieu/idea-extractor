package embeddings

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type OllamaEmbeddingRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type OllamaEmbeddingResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
}

// GenerateEmbedding takes problem title and description, concatenates them,
// and uses the Ollama API to generate a vector embedding.
func GenerateEmbedding(ctx context.Context, inputText string) ([]float32, error) {
	reqBody := OllamaEmbeddingRequest{
		Model: "embeddinggemma", // Use the same model as in analysis.go
		Input: inputText,
	}

	raw, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Ollama embedding request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx,
		"POST",
		"http://localhost:11434/api/embed", // Use the same Ollama endpoint as in analysis.go
		bytes.NewBuffer(raw),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Ollama embedding request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Ollama embedding API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody bytes.Buffer
		errBody.ReadFrom(resp.Body)
		return nil, fmt.Errorf("Ollama embedding API error (status %d): %s", resp.StatusCode, errBody.String())
	}

	var parsed OllamaEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("failed to decode Ollama embedding response: %w", err)
	}

	if len(parsed.Embeddings) == 0 || len(parsed.Embeddings[0]) == 0 {
		return nil, fmt.Errorf("empty embedding returned from Ollama")
	}

	return parsed.Embeddings[0], nil
}
