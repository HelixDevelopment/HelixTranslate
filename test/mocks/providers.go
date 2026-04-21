package mocks

import (
	"context"
	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/translator"
	"github.com/stretchr/testify/mock"
)

// MockTranslator implements translator interface for testing
type MockTranslator struct {
	mock.Mock
}

func (m *MockTranslator) GetName() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockTranslator) Translate(ctx context.Context, text, context string) (string, error) {
	args := m.Called(ctx, text, context)
	return args.String(0), args.Error(1)
}

func (m *MockTranslator) TranslateWithProgress(ctx context.Context, text, context string, eventBus *events.EventBus, sessionID string) (string, error) {
	args := m.Called(ctx, text, context, eventBus, sessionID)
	return args.String(0), args.Error(1)
}

func (m *MockTranslator) GetStats() translator.TranslationStats {
	args := m.Called()
	return args.Get(0).(translator.TranslationStats)
}

// MockLLMProvider implements LLM provider interface for testing
type MockLLMProvider struct {
	mock.Mock
}

func (m *MockLLMProvider) Translate(ctx context.Context, text, from, to string) (string, error) {
	args := m.Called(ctx, text, from, to)
	return args.String(0), args.Error(1)
}

func (m *MockLLMProvider) GetProvider() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockLLMProvider) IsAvailable(ctx context.Context) bool {
	args := m.Called(ctx)
	return args.Bool(0)
}

// MockDatabase implements database interface for testing
type MockDatabase struct {
	mock.Mock
}

func (m *MockDatabase) SaveTranslation(source, target, sourceLang, targetLang string) error {
	args := m.Called(source, target, sourceLang, targetLang)
	return args.Error(0)
}

func (m *MockDatabase) GetTranslation(source, sourceLang, targetLang string) (string, error) {
	args := m.Called(source, sourceLang, targetLang)
	return args.String(0), args.Error(1)
}

func (m *MockDatabase) Close() error {
	args := m.Called()
	return args.Error(0)
}

// MockSecurityProvider implements security interface for testing
type MockSecurityProvider struct {
	mock.Mock
}

func (m *MockSecurityProvider) Authenticate(token string) (bool, error) {
	args := m.Called(token)
	return args.Bool(0), args.Error(1)
}

func (m *MockSecurityProvider) Authorize(user, resource string) (bool, error) {
	args := m.Called(user, resource)
	return args.Bool(0), args.Error(1)
}

// MockStorage implements storage interface for testing
type MockStorage struct {
	mock.Mock
}

func (m *MockStorage) Save(key string, data []byte) error {
	args := m.Called(key, data)
	return args.Error(0)
}

func (m *MockStorage) Load(key string) ([]byte, error) {
	args := m.Called(key)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockStorage) Delete(key string) error {
	args := m.Called(key)
	return args.Error(0)
}

// MockProgressReporter implements progress reporting for testing
type MockProgressReporter struct {
	mock.Mock
}

func (m *MockProgressReporter) ReportProgress(current, total int, message string) {
	m.Called(current, total, message)
}

func (m *MockProgressReporter) ReportComplete(message string) {
	m.Called(message)
}

func (m *MockProgressReporter) ReportError(err error) {
	m.Called(err)
}
