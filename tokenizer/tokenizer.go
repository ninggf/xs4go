package tokenizer

import "strings"

// Tokenizer used by indexer and searcher
type Tokenizer interface {
	GetTokens(text string) []string
}

// DefaultTokenizer split text by space
type DefaultTokenizer struct {
	Name string
}

// GetTokens split text by space
func (tokenizer DefaultTokenizer) GetTokens(text string) []string {
	return strings.Split(text, " ")
}
