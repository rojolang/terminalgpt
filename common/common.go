package common

import (
	"fmt"
	"github.com/rojolang/terminalgpt/azure"
	"github.com/rojolang/terminalgpt/config"
	"github.com/rojolang/terminalgpt/gpt"
	"github.com/rojolang/terminalgpt/helpers"
)

func GenerateCompletion(cfg *config.Config, userMessage string) (string, int, int, int, int, error) {
	if cfg.AIProvider == "azure" {

		// Load the history
		history, err := helpers.LoadHistory(config.HistoryFile)
		if err != nil {
			return "", 0, 0, 0, 0, fmt.Errorf("failed to load history: %w", err)
		}

		// Pass the history to azure.GenerateCompletion
		return azure.GenerateCompletion(userMessage, cfg.SystemMessage, cfg.AzureURL, cfg.AzureAuthKey, cfg.ModelName, int32(cfg.MaxResponseTokens), float32(cfg.TopP), float32(cfg.Temperature), float32(cfg.FrequencyPenalty), float32(cfg.PresencePenalty), 20, history)
	}

	gptInstance, err := gpt.New(cfg)
	if err != nil {
		return "", 0, 0, 0, 0, fmt.Errorf("failed to create GPT instance: %w", err)
	}

	return gptInstance.GenerateCompletion(userMessage)
}
