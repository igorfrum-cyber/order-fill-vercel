package matching

import "context"

type Item struct {
	ID      string
	Article string
	Name    string
}

type Result struct {
	BlankID  string
	SourceID string
	Category string
	Score    float64
}

type Client interface {
	Match(ctx context.Context, mode string, blank, source []Item) ([]Result, error)
}
