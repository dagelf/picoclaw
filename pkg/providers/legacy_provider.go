// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package providers

import (
	"fmt"
	"strings"

	"github.com/sipeed/picoclaw/pkg/config"
)

// CreateProvider creates a provider based on the configuration.
// It uses the model_list configuration (new format) to create providers.
// The old providers config is automatically converted to model_list during config loading.
// Returns the provider, the model ID to use, and any error.
func CreateProvider(cfg *config.Config) (LLMProvider, string, error) {
	model := cfg.Agents.Defaults.GetModelName()

	// Ensure model_list is populated from providers config if needed
	// This handles two cases:
	// 1. ModelList is empty - convert all providers
	// 2. ModelList has some entries but not all providers - merge missing ones
	if cfg.HasProvidersConfig() {
		providerModels := config.ConvertProvidersToModelList(cfg)
		existingModelNames := make(map[string]bool)
		for _, m := range cfg.ModelList {
			existingModelNames[m.ModelName] = true
		}
		for _, pm := range providerModels {
			if !existingModelNames[pm.ModelName] {
				cfg.ModelList = append(cfg.ModelList, pm)
			}
		}
	}

	// Must have model_list at this point
	if len(cfg.ModelList) == 0 {
		return nil, "", fmt.Errorf("no providers configured. Please add entries to model_list in your config")
	}

	// If no default model is configured, pick the first usable model from model_list.
	// This keeps legacy/provider-only configs working without requiring explicit model_name.
	if strings.TrimSpace(model) == "" {
		model = pickImplicitModelName(cfg.ModelList)
		if model == "" {
			return nil, "", fmt.Errorf("no default model configured. Set agents.defaults.model_name (or agents.defaults.model) to a model_name from model_list")
		}
	}

	// Get model config from model_list
	modelCfg, err := cfg.GetModelConfig(model)
	if err != nil {
		return nil, "", fmt.Errorf("model %q not found in model_list: %w", model, err)
	}

	// Inject global workspace if not set in model config
	if modelCfg.Workspace == "" {
		modelCfg.Workspace = cfg.WorkspacePath()
	}

	// Use factory to create provider
	provider, modelID, err := CreateProviderFromConfig(modelCfg)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create provider for model %q: %w", model, err)
	}

	return provider, modelID, nil
}

func pickImplicitModelName(modelList []config.ModelConfig) string {
	for i := range modelList {
		if isImplicitlyUsableModel(modelList[i]) {
			return modelList[i].ModelName
		}
	}
	return ""
}

func isImplicitlyUsableModel(modelCfg config.ModelConfig) bool {
	if strings.TrimSpace(modelCfg.ModelName) == "" || strings.TrimSpace(modelCfg.Model) == "" {
		return false
	}

	if strings.TrimSpace(modelCfg.APIKey) != "" ||
		strings.TrimSpace(modelCfg.AuthMethod) != "" ||
		strings.TrimSpace(modelCfg.ConnectMode) != "" {
		return true
	}

	protocol, _ := ExtractProtocol(modelCfg.Model)
	switch protocol {
	case "antigravity", "claude-cli", "claudecli", "codex-cli", "codexcli", "github-copilot", "copilot":
		return true
	default:
		return false
	}
}
