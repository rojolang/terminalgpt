package azure

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/sdk/ai/azopenai"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/rojolang/terminalgpt/helpers"
	"github.com/sirupsen/logrus"
	"io"
	"time"
)

const LanguageModel = "gpt-4"

func GenerateCompletion(userMessage, systemMessage, azureURL, azureAuthKey, modelName string, maxTokens int32, topP, temperature, frequencyPenalty, presencePenalty float32, timeout time.Duration, history []helpers.HistoryEntry) (string, int, int, int, int, error) {
	userMessageTokens, err := helpers.CountTokens(userMessage, LanguageModel)
	if err != nil {
		return "", 0, 0, 0, 0, err
	}

	systemMessageTokens, err := helpers.CountTokens(systemMessage, LanguageModel)
	if err != nil {
		return "", 0, 0, 0, 0, err
	}

	historyTokens := 0
	for _, entry := range history {
		count, err := helpers.CountTokens(entry.Content, LanguageModel)
		if err != nil {
			return "", 0, 0, 0, 0, err
		}
		historyTokens += count
	}
	ctx := context.Background()

	keyCredential, err := azopenai.NewKeyCredential(azureAuthKey)
	if err != nil {
		logrus.WithError(err).Error("Failed to create key credential")
		return "", 0, 0, 0, 0, err
	}

	client, err := azopenai.NewClientWithKeyCredential(azureURL, keyCredential, nil)
	if err != nil {
		logrus.WithError(err).Error("Failed to create client with key credential")
		return "", 0, 0, 0, 0, err
	}

	messages := []azopenai.ChatMessage{
		{Role: to.Ptr(azopenai.ChatRoleSystem), Content: to.Ptr(systemMessage)},
		{Role: to.Ptr(azopenai.ChatRoleUser), Content: to.Ptr(userMessage)},
	}

	for _, entry := range history {
		messages = append([]azopenai.ChatMessage{
			{Role: to.Ptr(azopenai.ChatRole(entry.Role)), Content: to.Ptr(entry.Content)},
		}, messages...)
	}

	resp, err := client.GetChatCompletionsStream(ctx, azopenai.ChatCompletionsOptions{
		Messages:         messages,
		N:                to.Ptr[int32](1),
		Deployment:       modelName,
		Temperature:      to.Ptr(temperature),
		TopP:             to.Ptr(topP),
		MaxTokens:        to.Ptr(maxTokens),
		FrequencyPenalty: to.Ptr(frequencyPenalty),
		PresencePenalty:  to.Ptr(presencePenalty),
	}, nil)
	if err != nil {
		logrus.WithError(err).Error("Failed to get chat completions stream")
		return "", 0, 0, 0, 0, err
	}
	defer resp.ChatCompletionsStream.Close()

	responseTokens := 0

	for {
		_, cancel := context.WithTimeout(ctx, timeout)
		chatCompletions, err := resp.ChatCompletionsStream.Read()
		cancel()
		if err == io.EOF {
			break
		}
		if err != nil {
			logrus.WithError(err).Error("Failed to read from chat completions stream")
			return "", 0, 0, 0, 0, err
		}

		for _, choice := range chatCompletions.Choices {
			text := ""
			if choice.Delta.Content != nil {
				text = *choice.Delta.Content
			}
			if text == "" {
				continue
			}
			print(text)
			tokens, err := helpers.CountTokens(text, LanguageModel)
			if err != nil {
				return "", 0, 0, 0, 0, err
			}
			responseTokens += tokens
		}
	}

	return "", userMessageTokens, systemMessageTokens, responseTokens, historyTokens, nil
}
