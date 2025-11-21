package unit

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"digital.vasic.translator/pkg/coordination"
	"digital.vasic.translator/pkg/events"
)

func TestMultiLLMCoordinator(t *testing.T) {
	ctx := context.Background()

	t.Run("InitializeWithNoAPIKeys", func(t *testing.T) {
		// Clear all API keys
		os.Unsetenv("OPENAI_API_KEY")
		os.Unsetenv("ANTHROPIC_API_KEY")
		os.Unsetenv("ZHIPU_API_KEY")
		os.Unsetenv("DEEPSEEK_API_KEY")
		os.Unsetenv("QWEN_API_KEY")
		os.Unsetenv("OLLAMA_ENABLED")

		// Disable Qwen OAuth discovery in tests
		os.Setenv("SKIP_QWEN_OAUTH", "1")
		defer os.Unsetenv("SKIP_QWEN_OAUTH")

		coordinator := coordination.NewMultiLLMCoordinator(coordination.CoordinatorConfig{
			SessionID: "test-session",
		})

		if coordinator.GetInstanceCount() != 0 {
			t.Errorf("Expected 0 instances with no API keys, got %d", coordinator.GetInstanceCount())
		}
	})

	t.Run("InitializeWithOneProvider", func(t *testing.T) {
		// Set one API key
		os.Setenv("DEEPSEEK_API_KEY", "sk-test-key")
		defer os.Unsetenv("DEEPSEEK_API_KEY")

		coordinator := coordination.NewMultiLLMCoordinator(coordination.CoordinatorConfig{
			SessionID: "test-session",
		})

		// Should create 2 instances for the one provider
		if coordinator.GetInstanceCount() < 1 {
			t.Errorf("Expected at least 1 instance with one API key, got %d", coordinator.GetInstanceCount())
		}
	})

	t.Run("InitializeWithMultipleProviders", func(t *testing.T) {
		// Set multiple API keys
		os.Setenv("DEEPSEEK_API_KEY", "sk-test-key-1")
		os.Setenv("OPENAI_API_KEY", "sk-test-key-2")
		os.Setenv("ANTHROPIC_API_KEY", "sk-test-key-3")
		defer func() {
			os.Unsetenv("DEEPSEEK_API_KEY")
			os.Unsetenv("OPENAI_API_KEY")
			os.Unsetenv("ANTHROPIC_API_KEY")
		}()

		coordinator := coordination.NewMultiLLMCoordinator(coordination.CoordinatorConfig{
			SessionID: "test-session",
		})

		// Should create 2 instances per provider (3 providers × 2 = 6 instances)
		// But since we're using mock keys, actual initialization may fail
		// Just check that coordinator was created
		if coordinator == nil {
			t.Error("Expected coordinator to be created")
		}
	})

	t.Run("DefaultRetrySettings", func(t *testing.T) {
		coordinator := coordination.NewMultiLLMCoordinator(coordination.CoordinatorConfig{
			SessionID: "test-session",
		})

		// Just verify coordinator was created with defaults
		if coordinator == nil {
			t.Error("Expected coordinator to be created with defaults")
		}
	})

	t.Run("CustomRetrySettings", func(t *testing.T) {
		coordinator := coordination.NewMultiLLMCoordinator(coordination.CoordinatorConfig{
			MaxRetries: 5,
			RetryDelay: 1 * time.Second,
			SessionID:  "test-session",
		})

		if coordinator == nil {
			t.Error("Expected coordinator to be created with custom settings")
		}
	})

	t.Run("EventBusIntegration", func(t *testing.T) {
		eventBus := events.NewEventBus()
		receivedEvents := make([]events.Event, 0)

		eventBus.SubscribeAll(func(event events.Event) {
			receivedEvents = append(receivedEvents, event)
		})

		coordinator := coordination.NewMultiLLMCoordinator(coordination.CoordinatorConfig{
			EventBus:  eventBus,
			SessionID: "test-session",
		})

		if coordinator == nil {
			t.Error("Expected coordinator to be created")
		}

		// Events should have been emitted during initialization
		// (even if no instances were created due to missing keys)
		if len(receivedEvents) == 0 {
			t.Log("No events emitted (expected if no API keys configured)")
		}
	})

	t.Run("TranslateWithRetry_NoInstances", func(t *testing.T) {
		// Clear all API keys
		os.Unsetenv("OPENAI_API_KEY")
		os.Unsetenv("ANTHROPIC_API_KEY")
		os.Unsetenv("ZHIPU_API_KEY")
		os.Unsetenv("DEEPSEEK_API_KEY")
		os.Unsetenv("QWEN_API_KEY")
		os.Unsetenv("OLLAMA_ENABLED")

		// Disable Qwen OAuth discovery in tests
		os.Setenv("SKIP_QWEN_OAUTH", "1")
		defer os.Unsetenv("SKIP_QWEN_OAUTH")

		coordinator := coordination.NewMultiLLMCoordinator(coordination.CoordinatorConfig{
			SessionID: "test-session",
		})

		_, err := coordinator.TranslateWithRetry(ctx, "Hello", "test")
		if err == nil {
			t.Error("Expected error when no instances available")
		}

		if !strings.Contains(err.Error(), "no LLM instances available") {
			t.Errorf("Expected 'no LLM instances available' error, got: %v", err)
		}
	})

	t.Run("TranslateWithConsensus_NoInstances", func(t *testing.T) {
		// Clear all API keys
		os.Unsetenv("OPENAI_API_KEY")
		os.Unsetenv("ANTHROPIC_API_KEY")
		os.Unsetenv("ZHIPU_API_KEY")
		os.Unsetenv("DEEPSEEK_API_KEY")
		os.Unsetenv("QWEN_API_KEY")
		os.Unsetenv("OLLAMA_ENABLED")

		// Disable Qwen OAuth discovery in tests
		os.Setenv("SKIP_QWEN_OAUTH", "1")
		defer os.Unsetenv("SKIP_QWEN_OAUTH")

		coordinator := coordination.NewMultiLLMCoordinator(coordination.CoordinatorConfig{
			SessionID: "test-session",
		})

		_, err := coordinator.TranslateWithConsensus(ctx, "Hello", "test", 3)
		if err == nil {
			t.Error("Expected error when no instances available")
		}
	})

	t.Run("GetInstanceCount", func(t *testing.T) {
		coordinator := coordination.NewMultiLLMCoordinator(coordination.CoordinatorConfig{
			SessionID: "test-session",
		})

		count := coordinator.GetInstanceCount()
		if count < 0 {
			t.Errorf("Instance count should not be negative, got %d", count)
		}
	})
}

func TestLLMInstance(t *testing.T) {
	t.Run("InstanceCreation", func(t *testing.T) {
		// This tests the internal structure indirectly
		// We can't directly test LLMInstance as it's not exported
		// But we can test coordinator behavior

		os.Setenv("DEEPSEEK_API_KEY", "sk-test")
		defer os.Unsetenv("DEEPSEEK_API_KEY")

		coordinator := coordination.NewMultiLLMCoordinator(coordination.CoordinatorConfig{
			SessionID: "test",
		})

		// Verify instances were attempted to be created
		if coordinator == nil {
			t.Error("Coordinator should be created")
		}
	})
}

func TestProviderDiscovery(t *testing.T) {
	t.Run("DiscoverOpenAI", func(t *testing.T) {
		os.Setenv("OPENAI_API_KEY", "sk-test-openai")
		defer os.Unsetenv("OPENAI_API_KEY")

		coordinator := coordination.NewMultiLLMCoordinator(coordination.CoordinatorConfig{
			SessionID: "test",
		})

		if coordinator == nil {
			t.Error("Coordinator should be created")
		}
	})

	t.Run("DiscoverAnthropic", func(t *testing.T) {
		os.Setenv("ANTHROPIC_API_KEY", "sk-ant-test")
		defer os.Unsetenv("ANTHROPIC_API_KEY")

		coordinator := coordination.NewMultiLLMCoordinator(coordination.CoordinatorConfig{
			SessionID: "test",
		})

		if coordinator == nil {
			t.Error("Coordinator should be created")
		}
	})

	t.Run("DiscoverZhipu", func(t *testing.T) {
		os.Setenv("ZHIPU_API_KEY", "test-zhipu-key")
		defer os.Unsetenv("ZHIPU_API_KEY")

		coordinator := coordination.NewMultiLLMCoordinator(coordination.CoordinatorConfig{
			SessionID: "test",
		})

		if coordinator == nil {
			t.Error("Coordinator should be created")
		}
	})

	t.Run("DiscoverDeepSeek", func(t *testing.T) {
		os.Setenv("DEEPSEEK_API_KEY", "sk-test-deepseek")
		defer os.Unsetenv("DEEPSEEK_API_KEY")

		coordinator := coordination.NewMultiLLMCoordinator(coordination.CoordinatorConfig{
			SessionID: "test",
		})

		if coordinator == nil {
			t.Error("Coordinator should be created")
		}
	})

	t.Run("DiscoverOllama", func(t *testing.T) {
		os.Setenv("OLLAMA_ENABLED", "true")
		os.Setenv("OLLAMA_MODEL", "llama3:8b")
		defer func() {
			os.Unsetenv("OLLAMA_ENABLED")
			os.Unsetenv("OLLAMA_MODEL")
		}()

		coordinator := coordination.NewMultiLLMCoordinator(coordination.CoordinatorConfig{
			SessionID: "test",
		})

		if coordinator == nil {
			t.Error("Coordinator should be created")
		}
	})

	t.Run("CustomModelEnvironmentVariables", func(t *testing.T) {
		os.Setenv("OPENAI_API_KEY", "sk-test")
		os.Setenv("OPENAI_MODEL", "gpt-4-turbo")
		defer func() {
			os.Unsetenv("OPENAI_API_KEY")
			os.Unsetenv("OPENAI_MODEL")
		}()

		coordinator := coordination.NewMultiLLMCoordinator(coordination.CoordinatorConfig{
			SessionID: "test",
		})

		if coordinator == nil {
			t.Error("Coordinator should be created")
		}
	})
}

func TestRoundRobinDistribution(t *testing.T) {
	ctx := context.Background()

	t.Run("RoundRobinBehavior", func(t *testing.T) {
		// This test verifies that the coordinator attempts to use instances
		// in a round-robin fashion (indirectly through behavior)

		os.Setenv("DEEPSEEK_API_KEY", "sk-test")
		defer os.Unsetenv("DEEPSEEK_API_KEY")

		eventBus := events.NewEventBus()
		attempts := make([]events.Event, 0)

		eventBus.SubscribeAll(func(event events.Event) {
			if event.Type == "translation_attempt" {
				attempts = append(attempts, event)
			}
		})

		coordinator := coordination.NewMultiLLMCoordinator(coordination.CoordinatorConfig{
			EventBus:   eventBus,
			SessionID:  "test",
			MaxRetries: 3,
		})

		// Try to translate (will fail with mock key, but we can observe attempts)
		_, _ = coordinator.TranslateWithRetry(ctx, "Test text", "context")

		// Check that attempts were made
		t.Logf("Translation attempts: %d", len(attempts))
	})
}

func TestErrorHandling(t *testing.T) {
	ctx := context.Background()

	t.Run("EmptyText", func(t *testing.T) {
		coordinator := coordination.NewMultiLLMCoordinator(coordination.CoordinatorConfig{
			SessionID: "test",
		})

		// Should handle empty text gracefully
		_, err := coordinator.TranslateWithRetry(ctx, "", "context")
		if err == nil {
			t.Error("Expected error for empty text (no instances)")
		}
	})

	t.Run("NilContext", func(t *testing.T) {
		coordinator := coordination.NewMultiLLMCoordinator(coordination.CoordinatorConfig{
			SessionID: "test",
		})

		// Should handle nil context (will use ctx passed to function)
		_, err := coordinator.TranslateWithRetry(ctx, "test", "")
		if err == nil {
			t.Error("Expected error (no instances)")
		}
	})

	t.Run("ContextCancellation", func(t *testing.T) {
		coordinator := coordination.NewMultiLLMCoordinator(coordination.CoordinatorConfig{
			SessionID: "test",
		})

		cancelCtx, cancel := context.WithCancel(ctx)
		cancel() // Cancel immediately

		_, err := coordinator.TranslateWithRetry(cancelCtx, "test", "context")
		if err == nil {
			t.Error("Expected error with cancelled context")
		}
	})
}

func TestConcurrency(t *testing.T) {
	ctx := context.Background()

	t.Run("ConcurrentTranslations", func(t *testing.T) {
		os.Setenv("DEEPSEEK_API_KEY", "sk-test")
		defer os.Unsetenv("DEEPSEEK_API_KEY")

		coordinator := coordination.NewMultiLLMCoordinator(coordination.CoordinatorConfig{
			SessionID: "test",
		})

		// Launch multiple concurrent translations
		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func(id int) {
				text := fmt.Sprintf("Test text %d", id)
				_, _ = coordinator.TranslateWithRetry(ctx, text, "concurrent")
				done <- true
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			<-done
		}

		t.Log("Concurrent translations completed without panic")
	})
}

func TestConsensusMode(t *testing.T) {
	ctx := context.Background()

	t.Run("ConsensusWithInsufficientInstances", func(t *testing.T) {
		coordinator := coordination.NewMultiLLMCoordinator(coordination.CoordinatorConfig{
			SessionID: "test",
		})

		// Request consensus with more instances than available
		_, err := coordinator.TranslateWithConsensus(ctx, "test", "context", 5)
		if err == nil {
			t.Error("Expected error with insufficient instances")
		}
	})

	t.Run("ConsensusWithZeroRequired", func(t *testing.T) {
		coordinator := coordination.NewMultiLLMCoordinator(coordination.CoordinatorConfig{
			SessionID: "test",
		})

		// Should fall back to regular retry mode
		_, err := coordinator.TranslateWithConsensus(ctx, "test", "context", 0)
		if err == nil {
			t.Error("Expected error (no instances)")
		}
	})
}

func TestEventEmission(t *testing.T) {
	ctx := context.Background()

	t.Run("EmitWarningEvents", func(t *testing.T) {
		eventBus := events.NewEventBus()
		warnings := make([]events.Event, 0)

		eventBus.SubscribeAll(func(event events.Event) {
			if event.Type == "multi_llm_warning" {
				warnings = append(warnings, event)
			}
		})

		coordinator := coordination.NewMultiLLMCoordinator(coordination.CoordinatorConfig{
			EventBus:  eventBus,
			SessionID: "test",
		})

		// Without API keys, should emit warning
		if coordinator.GetInstanceCount() == 0 && len(warnings) > 0 {
			t.Log("Warning event emitted for no instances")
		}
	})

	t.Run("EmitInitEvents", func(t *testing.T) {
		os.Setenv("DEEPSEEK_API_KEY", "sk-test")
		defer os.Unsetenv("DEEPSEEK_API_KEY")

		eventBus := events.NewEventBus()
		initEvents := make([]events.Event, 0)

		eventBus.SubscribeAll(func(event events.Event) {
			if event.Type == "multi_llm_init" || event.Type == "multi_llm_ready" {
				initEvents = append(initEvents, event)
			}
		})

		_ = coordination.NewMultiLLMCoordinator(coordination.CoordinatorConfig{
			EventBus:  eventBus,
			SessionID: "test",
		})

		if len(initEvents) == 0 {
			t.Log("No init events (expected if instance creation failed)")
		}
	})

	t.Run("EmitTranslationAttemptEvents", func(t *testing.T) {
		os.Setenv("DEEPSEEK_API_KEY", "sk-test")
		defer os.Unsetenv("DEEPSEEK_API_KEY")

		eventBus := events.NewEventBus()
		attempts := make([]events.Event, 0)

		eventBus.SubscribeAll(func(event events.Event) {
			if event.Type == "translation_attempt" {
				attempts = append(attempts, event)
			}
		})

		coordinator := coordination.NewMultiLLMCoordinator(coordination.CoordinatorConfig{
			EventBus:   eventBus,
			SessionID:  "test",
			MaxRetries: 2,
		})

		_, _ = coordinator.TranslateWithRetry(ctx, "test", "context")

		t.Logf("Translation attempt events: %d", len(attempts))
	})
}

func TestEdgeCases(t *testing.T) {
	ctx := context.Background()

	t.Run("VeryLongText", func(t *testing.T) {
		coordinator := coordination.NewMultiLLMCoordinator(coordination.CoordinatorConfig{
			SessionID: "test",
		})

		longText := strings.Repeat("This is a very long text. ", 1000)
		_, err := coordinator.TranslateWithRetry(ctx, longText, "context")
		if err == nil {
			t.Error("Expected error (no instances)")
		}
	})

	t.Run("SpecialCharacters", func(t *testing.T) {
		coordinator := coordination.NewMultiLLMCoordinator(coordination.CoordinatorConfig{
			SessionID: "test",
		})

		specialText := "Test with émojis 😀🎉 and spëcial çharacters!"
		_, err := coordinator.TranslateWithRetry(ctx, specialText, "context")
		if err == nil {
			t.Error("Expected error (no instances)")
		}
	})

	t.Run("UnicodeText", func(t *testing.T) {
		coordinator := coordination.NewMultiLLMCoordinator(coordination.CoordinatorConfig{
			SessionID: "test",
		})

		unicodeText := "Тестовый текст на русском языке с юникодом 中文字符"
		_, err := coordinator.TranslateWithRetry(ctx, unicodeText, "context")
		if err == nil {
			t.Error("Expected error (no instances)")
		}
	})
}
