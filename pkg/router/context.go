package router

import "context"

type contextKey string

const paramsKey contextKey = "nexgo_params"

// WithParams stores route params in context
func WithParams(ctx context.Context, params map[string]string) context.Context {
	return context.WithValue(ctx, paramsKey, params)
}

// Param retrieves a route parameter from the request context
func Param(r interface{ Context() context.Context }, name string) string {
	params, ok := r.Context().Value(paramsKey).(map[string]string)
	if !ok {
		return ""
	}
	return params[name]
}

// Params retrieves all route parameters
func Params(r interface{ Context() context.Context }) map[string]string {
	params, ok := r.Context().Value(paramsKey).(map[string]string)
	if !ok {
		return map[string]string{}
	}
	return params
}
