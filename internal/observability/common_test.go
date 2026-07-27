package observability

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
)

// Test_newResource проверяет общие OTEL-метаданные (атрибуты) сервиса.
func Test_newResource(t *testing.T) {
	// Arrange
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "deployment.environment=test")

	// Act
	resource, err := newResource(context.Background(), "goph-profile-test")

	// Assert
	require.NoError(t, err)
	attrs := resource.Set()

	serviceName, ok := attrs.Value(attribute.Key("service.name"))
	require.True(t, ok)
	assert.Equal(t, "goph-profile-test", serviceName.AsString())

	environment, ok := attrs.Value(attribute.Key("deployment.environment"))
	require.True(t, ok)
	assert.Equal(t, "test", environment.AsString())
}
