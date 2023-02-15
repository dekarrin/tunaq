package types

// TokenStream is a stream of tokens read from source text. The stream may be
// lazily-loaded or immediately available.
type TokenStream interface {
	// Next returns the next token in the stream and advances the stream by one
	// token.
	Next() Token

	// Peek returns the next token in the stream without advancing the stream.
	Peek() Token

	// HasNext returns whether the stream has any additional tokens.
	HasNext() bool
}
