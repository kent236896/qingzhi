package weknora

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

const memoryConfigKVKey = "memory-config"

// MemoryConfig is the workspace-level long-term memory configuration.
type MemoryConfig struct {
	Enabled                   bool   `json:"enabled"`
	WriteMode                 string `json:"write_mode"`
	ExtractModelID            string `json:"extract_model_id"`
	MaxItems                  int    `json:"max_items"`
	ExtractDelaySeconds       int    `json:"extract_delay_seconds"`
	ExtractMinIntervalSeconds int    `json:"extract_min_interval_seconds"`
	ExtractInstructions       string `json:"extract_instructions"`
	InterestThreshold         int    `json:"interest_threshold"`
	EmbeddingModelID          string `json:"embedding_model_id"`
	VectorRecall              bool   `json:"vector_recall"`
	RetrievalConditioning     bool   `json:"retrieval_conditioning"`
}

// DefaultMemoryConfig returns conservative defaults used when WeKnora returns empty data.
func DefaultMemoryConfig() MemoryConfig {
	return MemoryConfig{
		Enabled:                   false,
		WriteMode:                 "explicit_only",
		MaxItems:                  200,
		ExtractDelaySeconds:       30,
		ExtractMinIntervalSeconds: 60,
		InterestThreshold:         3,
		VectorRecall:              true,
		RetrievalConditioning:     true,
	}
}

// GetMemoryConfig reads GET /api/v1/tenants/kv/memory-config for the API key's space.
func (c *HTTPClient) GetMemoryConfig(ctx context.Context) (MemoryConfig, error) {
	client, _, _ := c.session()
	if client == nil {
		return MemoryConfig{}, fmt.Errorf("weknora client is not configured")
	}
	raw, err := client.GetTenantKV(ctx, memoryConfigKVKey)
	if err != nil {
		return MemoryConfig{}, wrap(err)
	}
	return decodeMemoryConfig(raw)
}

// UpdateMemoryConfig writes PUT /api/v1/tenants/kv/memory-config with a full object.
func (c *HTTPClient) UpdateMemoryConfig(ctx context.Context, cfg MemoryConfig) (MemoryConfig, error) {
	if err := ValidateMemoryConfig(cfg); err != nil {
		return MemoryConfig{}, err
	}
	client, _, _ := c.session()
	if client == nil {
		return MemoryConfig{}, fmt.Errorf("weknora client is not configured")
	}
	raw, err := client.UpdateTenantKV(ctx, memoryConfigKVKey, cfg)
	if err != nil {
		return MemoryConfig{}, wrap(err)
	}
	if len(raw) == 0 || string(raw) == "null" {
		return c.GetMemoryConfig(ctx)
	}
	return decodeMemoryConfig(raw)
}

func decodeMemoryConfig(raw json.RawMessage) (MemoryConfig, error) {
	cfg := DefaultMemoryConfig()
	if len(raw) == 0 || string(raw) == "null" {
		return cfg, nil
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return MemoryConfig{}, fmt.Errorf("parse memory-config: %w", err)
	}
	return cfg, nil
}

// MergeMemoryConfig overlays non-zero / provided fields onto current.
// Callers should pass a full form from the admin UI; this helper preserves
// unknown future fields by starting from current.
func MergeMemoryConfig(current MemoryConfig, patch MemoryConfig) MemoryConfig {
	out := current
	out.Enabled = patch.Enabled
	if patch.WriteMode != "" {
		out.WriteMode = patch.WriteMode
	}
	out.ExtractModelID = strings.TrimSpace(patch.ExtractModelID)
	if patch.MaxItems > 0 {
		out.MaxItems = patch.MaxItems
	}
	if patch.ExtractDelaySeconds >= 0 {
		out.ExtractDelaySeconds = patch.ExtractDelaySeconds
	}
	if patch.ExtractMinIntervalSeconds >= 0 {
		out.ExtractMinIntervalSeconds = patch.ExtractMinIntervalSeconds
	}
	out.ExtractInstructions = patch.ExtractInstructions
	if patch.InterestThreshold > 0 {
		out.InterestThreshold = patch.InterestThreshold
	}
	out.EmbeddingModelID = strings.TrimSpace(patch.EmbeddingModelID)
	out.VectorRecall = patch.VectorRecall
	out.RetrievalConditioning = patch.RetrievalConditioning
	return out
}

// ValidateMemoryConfig checks values before PUT.
func ValidateMemoryConfig(cfg MemoryConfig) error {
	mode := strings.TrimSpace(cfg.WriteMode)
	if mode != "explicit_only" && mode != "auto" {
		return fmt.Errorf("write_mode 仅支持 explicit_only 或 auto")
	}
	if cfg.MaxItems < 1 || cfg.MaxItems > 10000 {
		return fmt.Errorf("max_items 须在 1–10000")
	}
	if cfg.ExtractDelaySeconds < 0 || cfg.ExtractDelaySeconds > 3600 {
		return fmt.Errorf("extract_delay_seconds 须在 0–3600")
	}
	if cfg.ExtractMinIntervalSeconds < 0 || cfg.ExtractMinIntervalSeconds > 86400 {
		return fmt.Errorf("extract_min_interval_seconds 须在 0–86400")
	}
	if cfg.InterestThreshold < 1 || cfg.InterestThreshold > 100 {
		return fmt.Errorf("interest_threshold 须在 1–100")
	}
	if len([]rune(cfg.ExtractInstructions)) > 4000 {
		return fmt.Errorf("extract_instructions 过长")
	}
	return nil
}
