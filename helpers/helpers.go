// Package helpers provides a set of helper functions.
//
// Functions:
// - GetTokenCount: Returns the token count of a text string for a specific model.
package helpers

import (
	"fmt"

	tiktoken "github.com/pkoukk/tiktoken-go"
)

// GetTokenCount function returns the token count of a text string for a specific model.
// It takes a text string and a model name as input and returns a Tiktoken and an error.
func GetTokenCount(text string, modelName string) (*tiktoken.Tiktoken, error) {
	tkm, err := tiktoken.EncodingForModel(modelName)
	if err != nil {
		return nil, fmt.Errorf("EncodingForModel: %v", err)
	}
	return tkm, nil
}
