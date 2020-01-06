package middleware

import (
	"context"

	"github.com/containous/traefik/v2/pkg/server/internal"
)

// AddProviderInContext adds the provider name in the context.
func AddProviderInContext(ctx context.Context, elementName string) context.Context {
	return internal.AddProviderInContext(ctx, elementName)
}

// GetQualifiedName gets the fully qualified name.
func GetQualifiedName(ctx context.Context, elementName string) string {
	return internal.GetQualifiedName(ctx, elementName)
}

// MakeQualifiedName creates a qualified name for an element.
func MakeQualifiedName(providerName string, elementName string) string {
	return internal.MakeQualifiedName(providerName, elementName)
}
