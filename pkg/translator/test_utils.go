package translator

import (
	"context"

	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/language"
	"github.com/stretchr/testify/mock"
)

// MockTranslator is a mock implementation of the Translator interface
type MockTranslator struct {
	mock.Mock
}

func (m *MockTranslator) Translate(ctx context.Context, text string, context string) (string, error) {
	args := m.Called(ctx, text, context)
	return args.String(0), args.Error(1)
}

func (m *MockTranslator) TranslateWithProgress(ctx context.Context, text string, context string, eventBus *events.EventBus, sessionID string) (string, error) {
	args := m.Called(ctx, text, context, eventBus, sessionID)
	return args.String(0), args.Error(1)
}

func (m *MockTranslator) GetStats() TranslationStats {
	args := m.Called()
	return args.Get(0).(TranslationStats)
}

func (m *MockTranslator) GetName() string {
	args := m.Called()
	return args.String(0)
}

// MockLLMDetector implements the LLMDetector interface
type MockLLMDetector struct {
	mock.Mock
}

func (m *MockLLMDetector) DetectLanguage(ctx context.Context, text string) (string, error) {
	args := m.Called(ctx, text)
	return args.String(0), args.Error(1)
}

// NewMockDetector creates a mock detector that implements the required interface
func NewMockDetector() *language.Detector {
	mockLLMDetector := new(MockLLMDetector)
	return language.NewDetector(mockLLMDetector)
}
