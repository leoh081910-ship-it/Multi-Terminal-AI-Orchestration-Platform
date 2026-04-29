package connector

import (
	"context"
	"testing"
)

func TestConnectorRegistry(t *testing.T) {
	registry := NewConnectorRegistry()

	// Create a mock connector
	mockConnector := &mockConnector{connectorType: "test", version: "1.0.0"}

	// Test Register
	t.Run("Register new connector", func(t *testing.T) {
		err := registry.Register(mockConnector)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("Register duplicate connector type", func(t *testing.T) {
		err := registry.Register(mockConnector)
		if err == nil {
			t.Error("Expected error for duplicate registration")
		}
	})

	// Test Get
	t.Run("Get registered connector", func(t *testing.T) {
		connector, err := registry.Get("test")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if connector.GetConnectorType() != "test" {
			t.Errorf("Expected type 'test', got %s", connector.GetConnectorType())
		}
	})

	t.Run("Get unregistered connector", func(t *testing.T) {
		_, err := registry.Get("nonexistent")
		if err == nil {
			t.Error("Expected error for unregistered connector")
		}
	})

	// Test List
	t.Run("List registered connectors", func(t *testing.T) {
		types := registry.List()
		if len(types) != 1 {
			t.Errorf("Expected 1 connector type, got %d", len(types))
		}
		if types[0] != "test" {
			t.Errorf("Expected 'test', got %s", types[0])
		}
	})

	// Test HealthCheck
	t.Run("HealthCheck all connectors", func(t *testing.T) {
		results := registry.HealthCheck(context.Background())
		if len(results) != 1 {
			t.Errorf("Expected 1 health check result, got %d", len(results))
		}
		if results["test"] != nil {
			t.Errorf("Expected nil error for healthy connector, got %v", results["test"])
		}
	})
}

// mockConnector implements the Connector interface for testing
type mockConnector struct {
	connectorType string
	version       string
}

func (m *mockConnector) DiscoverTasks(ctx context.Context) ([]TaskCard, error) {
	return nil, nil
}

func (m *mockConnector) HydrateContext(ctx context.Context, taskID string, baseContext map[string]interface{}) (map[string]interface{}, error) {
	return baseContext, nil
}

func (m *mockConnector) AckResult(ctx context.Context, result TaskResult) error {
	return nil
}

func (m *mockConnector) WriteBackArtifacts(ctx context.Context, taskID string, artifacts []Artifact) error {
	return nil
}

func (m *mockConnector) GetConnectorType() string {
	return m.connectorType
}

func (m *mockConnector) GetConnectorVersion() string {
	return m.version
}

func (m *mockConnector) HealthCheck(ctx context.Context) error {
	return nil
}
