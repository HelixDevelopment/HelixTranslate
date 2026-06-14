package grpc

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"digital.vasic.translator/pkg/events"
	"digital.vasic.translator/pkg/grpc/proto"
	"digital.vasic.translator/pkg/logger"
)

// Server implements the gRPC TranslationService
type Server struct {
	proto.UnimplementedTranslationServiceServer

	// Core components
	eventBus   *events.EventBus
	logger     logger.Logger
	grpcServer *grpc.Server

	// Translation management
	translator    CoreTranslator
	sessions      map[string]*TranslationSession
	sessionsMutex sync.RWMutex

	// Event streaming
	streams      map[string]chan *proto.TranslationProgressEvent
	streamsMutex sync.RWMutex

	// Provider information
	providers *ProviderRegistry

	// Configuration
	config *ServerConfig
}

// ServerConfig holds server configuration
type ServerConfig struct {
	MaxConcurrentTranslations int
	SessionTimeout            time.Duration
	StreamBufferSize          int
	EnableMetrics             bool
}

// TranslationSession represents an active translation session
type TranslationSession struct {
	ID         string
	Status     string
	Request    *proto.TranslationRequest
	Response   *proto.TranslationStatusResponse
	CreatedAt  time.Time
	UpdatedAt  time.Time
	CancelFunc context.CancelFunc
	EventBus   *events.EventBus
	Logger     logger.Logger

	// Progress tracking
	CurrentStep string
	Progress    float64
	Steps       []*proto.TranslationStep
	Files       []*proto.GeneratedFile

	// Runtime context
	Ctx context.Context
}

// CoreTranslator interface for the actual translation engine
type CoreTranslator interface {
	Translate(ctx context.Context, req *proto.TranslationRequest, eventBus *events.EventBus) (*proto.TranslationStatusResponse, error)
	Cancel(sessionID string) error
	GetStatus(sessionID string) (*proto.TranslationStatusResponse, error)
}

// ProviderRegistry manages available translation providers
type ProviderRegistry struct {
	providers map[string]*proto.ProviderInfo
	mutex     sync.RWMutex
}

// NewServer creates a new gRPC server
func NewServer(eventBus *events.EventBus, logger logger.Logger, translator CoreTranslator, config *ServerConfig) *Server {
	if config == nil {
		config = &ServerConfig{
			MaxConcurrentTranslations: 10,
			SessionTimeout:            24 * time.Hour,
			StreamBufferSize:          100,
			EnableMetrics:             true,
		}
	}

	// Create gRPC server with interceptors
	grpcServer := grpc.NewServer(
		grpc.MaxRecvMsgSize(4*1024*1024), // 4MB max message size
		grpc.MaxSendMsgSize(4*1024*1024), // 4MB max message size
	)

	server := &Server{
		eventBus:   eventBus,
		logger:     logger,
		grpcServer: grpcServer,
		translator: translator,
		sessions:   make(map[string]*TranslationSession),
		streams:    make(map[string]chan *proto.TranslationProgressEvent),
		providers:  NewProviderRegistry(),
		config:     config,
	}

	// Start cleanup routine
	go server.cleanupRoutine()

	return server
}

// StartTranslation starts a new translation job
func (s *Server) StartTranslation(ctx context.Context, req *proto.TranslationRequest) (*proto.TranslationResponse, error) {
	// Validate the request BEFORE touching any of its sub-messages. In proto3
	// every message field (provider_config, options) is optional on the wire, so
	// a client can legitimately send a request with provider_config unset. The
	// old code dereferenced req.ProviderConfig.Type in the opening log call,
	// which panicked on a nil ProviderConfig — and grpc-go registers no panic
	// recovery here, so a single malformed request crashed the serving goroutine
	// (a remote-trigger DoS). Reject malformed input cleanly with InvalidArgument.
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "translation request is required")
	}
	if req.SessionId == "" {
		return nil, status.Error(codes.InvalidArgument, "session_id is required")
	}
	if req.ProviderConfig == nil {
		return &proto.TranslationResponse{
			SessionId: req.SessionId,
			Status:    "error",
			Message:   "provider_config is required",
		}, status.Error(codes.InvalidArgument, "provider_config is required")
	}

	s.logger.Info("Starting translation request", map[string]interface{}{
		"session_id": req.SessionId,
		"input_file": req.InputFile,
		"provider":   req.ProviderConfig.Type,
	})

	// Check session limits AND reserve the session id atomically under a single
	// write lock. The previous code checked the count under an RLock and inserted
	// under a separate Lock, so two concurrent requests could both pass the gate
	// (TOCTOU). It also overwrote an existing session with the same id WITHOUT
	// cancelling the prior session's CancelFunc — leaking that session's timeout
	// context (a live timer/goroutine) and orphaning its background translation.
	s.sessionsMutex.Lock()
	if _, dup := s.sessions[req.SessionId]; dup {
		s.sessionsMutex.Unlock()
		return &proto.TranslationResponse{
			SessionId: req.SessionId,
			Status:    "error",
			Message:   "Translation session already exists",
		}, status.Errorf(codes.AlreadyExists, "translation session already exists: %s", req.SessionId)
	}
	if len(s.sessions) >= s.config.MaxConcurrentTranslations {
		s.sessionsMutex.Unlock()
		return &proto.TranslationResponse{
			SessionId: req.SessionId,
			Status:    "error",
			Message:   "Maximum concurrent translations reached",
		}, status.Errorf(codes.ResourceExhausted, "maximum concurrent translations (%d) reached", s.config.MaxConcurrentTranslations)
	}

	// Create session context with timeout
	sessionCtx, cancel := context.WithTimeout(context.Background(), s.config.SessionTimeout)

	// Create translation session
	session := &TranslationSession{
		ID:         req.SessionId,
		Status:     "pending",
		Request:    req,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		CancelFunc: cancel,
		EventBus:   events.NewEventBus(), // Private event bus for this session
		Logger:     s.logger,
		Ctx:        sessionCtx,
		Steps:      make([]*proto.TranslationStep, 0),
		Files:      make([]*proto.GeneratedFile, 0),
	}

	// Store session (still under the same write lock — the gate above already
	// proved this id is free and the slot is available).
	s.sessions[req.SessionId] = session
	s.sessionsMutex.Unlock()

	// Start translation in goroutine
	go s.runTranslation(session)

	// Return initial response
	return &proto.TranslationResponse{
		SessionId:                req.SessionId,
		Status:                   "started",
		Message:                  "Translation started successfully",
		StartedAt:                timeToProto(time.Now()),
		EstimatedDurationSeconds: 300, // 5 minutes estimate
	}, nil
}

// GetTranslationStatus returns the current status of a translation
func (s *Server) GetTranslationStatus(ctx context.Context, req *proto.TranslationStatusRequest) (*proto.TranslationStatusResponse, error) {
	// Snapshot the session fields under the read lock into a LOCAL response. The
	// background runTranslation / CancelTranslation paths mutate these fields
	// under the same lock, so the snapshot is race-free. The slow
	// translator.GetStatus call below runs AFTER the lock is released (no I/O
	// under the lock). We deliberately do NOT write session.Response from this
	// read RPC — that was an unsynchronized shared write.
	s.sessionsMutex.RLock()
	session, exists := s.sessions[req.SessionId]
	if !exists {
		s.sessionsMutex.RUnlock()
		// Return a typed gRPC status so the client can switch on codes.NotFound.
		// A bare fmt.Errorf is mapped to codes.Unknown by grpc-go, which is
		// indistinguishable from a genuine server fault — breaking the error-code
		// contract the write paths (StartTranslation) already honour.
		return nil, status.Errorf(codes.NotFound, "translation session not found: %s", req.SessionId)
	}
	resp := &proto.TranslationStatusResponse{
		SessionId:          session.ID,
		Status:             session.Status,
		ProgressPercentage: session.Progress,
		CurrentStep:        session.CurrentStep,
		StartedAt:          timeToProto(session.CreatedAt),
		UpdatedAt:          timeToProto(session.UpdatedAt),
		Files:              session.Files,
		Steps:              session.Steps,
	}
	s.sessionsMutex.RUnlock()

	// Try to get status from core translator (no lock held — may do I/O).
	if coreStatus, err := s.translator.GetStatus(req.SessionId); err == nil {
		resp.Status = coreStatus.Status
		resp.ProgressPercentage = coreStatus.ProgressPercentage
		resp.CurrentStep = coreStatus.CurrentStep
		resp.EstimatedCompletion = coreStatus.EstimatedCompletion
		resp.Files = coreStatus.Files
		resp.Steps = coreStatus.Steps
	}

	return resp, nil
}

// ListTranslations returns all translation sessions
func (s *Server) ListTranslations(ctx context.Context, _ *emptypb.Empty) (*proto.TranslationListResponse, error) {
	// Snapshot session IDs under the read lock, then release it BEFORE calling
	// GetTranslationStatus (which takes the read lock itself). Holding the lock
	// across that call is a recursive read-lock that can deadlock if a writer
	// arrives between the two acquisitions.
	s.sessionsMutex.RLock()
	ids := make([]string, 0, len(s.sessions))
	for _, session := range s.sessions {
		ids = append(ids, session.ID)
	}
	s.sessionsMutex.RUnlock()

	translations := make([]*proto.TranslationStatusResponse, 0, len(ids))

	for _, id := range ids {
		status, err := s.GetTranslationStatus(ctx, &proto.TranslationStatusRequest{
			SessionId: id,
		})
		if err != nil {
			s.logger.Warn("Failed to get session status", map[string]interface{}{
				"session_id": id,
				"error":      err.Error(),
			})
			continue
		}
		translations = append(translations, status)
	}

	return &proto.TranslationListResponse{
		Translations: translations,
		TotalCount:   int32(len(translations)),
	}, nil
}

// CancelTranslation cancels a translation job
func (s *Server) CancelTranslation(ctx context.Context, req *proto.CancelTranslationRequest) (*proto.CancelTranslationResponse, error) {
	s.logger.Info("Cancelling translation", map[string]interface{}{
		"session_id": req.SessionId,
		"reason":     req.Reason,
	})

	s.sessionsMutex.RLock()
	session, exists := s.sessions[req.SessionId]
	s.sessionsMutex.RUnlock()

	if !exists {
		return &proto.CancelTranslationResponse{
			SessionId: req.SessionId,
			Success:   false,
			Message:   "Translation session not found",
		}, nil
	}

	// Cancel the session context
	if session.CancelFunc != nil {
		session.CancelFunc()
	}

	// Call core translator cancel
	if err := s.translator.Cancel(req.SessionId); err != nil {
		s.logger.Warn("Failed to cancel translation in core translator", map[string]interface{}{
			"session_id": req.SessionId,
			"error":      err.Error(),
		})
	}

	// Update session status under the lock (the fields are read concurrently by
	// GetTranslationStatus / ListTranslations).
	s.sessionsMutex.Lock()
	session.Status = "cancelled"
	session.UpdatedAt = time.Now()
	s.sessionsMutex.Unlock()

	// Emit cancellation event
	s.emitProgressEvent(session.ID, "cancelled", "", 0, "Translation cancelled: "+req.Reason, nil)

	return &proto.CancelTranslationResponse{
		SessionId: req.SessionId,
		Success:   true,
		Message:   "Translation cancelled successfully",
	}, nil
}

// StreamTranslationProgress streams translation progress events
func (s *Server) StreamTranslationProgress(req *proto.TranslationStreamRequest, stream proto.TranslationService_StreamTranslationProgressServer) error {
	s.logger.Info("Starting progress stream", map[string]interface{}{
		"session_id": req.SessionId,
		"client_id":  req.ClientId,
	})

	// Create event channel for this stream
	eventChan := make(chan *proto.TranslationProgressEvent, s.config.StreamBufferSize)

	// Store stream
	streamKey := fmt.Sprintf("%s:%s", req.SessionId, req.ClientId)
	s.streamsMutex.Lock()
	s.streams[streamKey] = eventChan
	s.streamsMutex.Unlock()

	// Clean up on exit
	defer func() {
		s.streamsMutex.Lock()
		delete(s.streams, streamKey)
		s.streamsMutex.Unlock()
		close(eventChan)
	}()

	// Send current status
	if currentStatus, err := s.GetTranslationStatus(stream.Context(), &proto.TranslationStatusRequest{
		SessionId: req.SessionId,
	}); err == nil {
		initialEvent := &proto.TranslationProgressEvent{
			SessionId:          req.SessionId,
			EventType:          "status_update",
			ProgressPercentage: currentStatus.ProgressPercentage,
			CurrentOperation:   currentStatus.CurrentStep,
			Message:            fmt.Sprintf("Current status: %s", currentStatus.Status),
			Timestamp:          timeToProto(time.Now()),
		}
		if err := stream.Send(initialEvent); err != nil {
			return err
		}
	}

	// Stream events
	for {
		select {
		case event, ok := <-eventChan:
			if !ok {
				return nil // Channel closed
			}

			if err := stream.Send(event); err != nil {
				return err
			}

		case <-stream.Context().Done():
			return stream.Context().Err()
		}
	}
}

// GetProviders returns available translation providers
func (s *Server) GetProviders(ctx context.Context, _ *emptypb.Empty) (*proto.ProvidersResponse, error) {
	providers := s.providers.GetAll()
	return &proto.ProvidersResponse{
		Providers: providers,
	}, nil
}

// SubscribeEvents subscribes to system events
func (s *Server) SubscribeEvents(req *proto.EventSubscriptionRequest, stream proto.TranslationService_SubscribeEventsServer) error {
	s.logger.Info("Starting event subscription", map[string]interface{}{
		"client_id":   req.ClientId,
		"event_types": req.EventTypes,
	})

	// Create event channel
	eventChan := make(chan *proto.SystemEvent, 100)

	// Subscribe to event bus
	subID := s.eventBus.SubscribeAll(func(event events.Event) {
		// Filter by event types if specified
		if len(req.EventTypes) > 0 {
			found := false
			for _, eventType := range req.EventTypes {
				if string(event.Type) == eventType {
					found = true
					break
				}
			}
			if !found {
				return
			}
		}

		// Convert data map
		data := make(map[string]string)
		for k, v := range event.Data {
			data[k] = fmt.Sprintf("%v", v)
		}

		// Convert to proto
		protoEvent := &proto.SystemEvent{
			EventType: string(event.Type),
			Timestamp: timeToProto(event.Timestamp),
			Data:      data,
			SessionId: event.SessionID,
			ClientId:  req.ClientId,
		}

		select {
		case eventChan <- protoEvent:
		default:
			// Channel full, drop event
		}
	})

	// Clean up on exit: remove our handler from the bus so it is not leaked and
	// invoked on every future Publish for the lifetime of the server. We
	// deliberately do NOT close(eventChan): the handler is removed first, but a
	// Publish already in flight (it snapshots handlers, then releases the lock)
	// could still invoke our handler after Unsubscribe returns — sending on a
	// closed channel would panic. The stream loop below exits via the request
	// context being cancelled, so the channel never needs closing; it is GC'd
	// once unreferenced.
	defer s.eventBus.Unsubscribe(subID)

	// Stream events
	for {
		select {
		case event, ok := <-eventChan:
			if !ok {
				return nil // Channel closed
			}

			if err := stream.Send(event); err != nil {
				return err
			}

		case <-stream.Context().Done():
			return stream.Context().Err()
		}
	}
}

// Internal methods

func (s *Server) runTranslation(session *TranslationSession) {
	s.logger.Info("Starting translation execution", map[string]interface{}{
		"session_id": session.ID,
	})

	// Update status under the lock (read concurrently by
	// GetTranslationStatus / ListTranslations).
	s.sessionsMutex.Lock()
	session.Status = "running"
	session.UpdatedAt = time.Now()
	s.sessionsMutex.Unlock()

	// Emit start event
	s.emitProgressEvent(session.ID, "started", "", 0, "Translation started", nil)

	// Run translation
	response, err := s.translator.Translate(session.Ctx, session.Request, session.EventBus)

	s.sessionsMutex.Lock()
	defer s.sessionsMutex.Unlock()

	if err != nil {
		session.Status = "failed"
		session.UpdatedAt = time.Now()

		s.logger.Error("Translation failed", map[string]interface{}{
			"session_id": session.ID,
			"error":      err.Error(),
		})

		// Emit error event
		s.emitProgressEvent(session.ID, "error", "", 0, fmt.Sprintf("Translation failed: %s", err.Error()), map[string]interface{}{
			"error": err.Error(),
		})

		return
	}

	// Update session with results
	session.Status = "completed"
	session.UpdatedAt = time.Now()
	session.Progress = 100.0
	session.Files = response.Files
	session.Steps = response.Steps

	s.logger.Info("Translation completed", map[string]interface{}{
		"session_id":      session.ID,
		"files_generated": len(response.Files),
	})

	// Emit completion event
	s.emitProgressEvent(session.ID, "completed", "", 100, "Translation completed successfully", map[string]interface{}{
		"files_count": len(response.Files),
		"duration":    time.Since(session.CreatedAt).String(),
	})
}

func (s *Server) emitProgressEvent(sessionID, eventType, stepName string, progress float64, message string, metadata map[string]interface{}) {
	event := &proto.TranslationProgressEvent{
		SessionId:          sessionID,
		EventType:          eventType,
		StepName:           stepName,
		ProgressPercentage: progress,
		Message:            message,
		Metadata:           convertMetadata(metadata),
		Timestamp:          timeToProto(time.Now()),
	}

	// Send to all active streams for this session
	s.streamsMutex.RLock()
	for streamKey, eventChan := range s.streams {
		if strings.HasPrefix(streamKey, sessionID+":") {
			select {
			case eventChan <- event:
			default:
				// Channel full, skip
			}
		}
	}
	s.streamsMutex.RUnlock()

	// Also emit to main event bus. events.NewEvent leaves Event.SessionID empty,
	// so we MUST stamp the sessionID we were handed before publishing — otherwise
	// SubscribeEvents maps every lifecycle event to a SystemEvent with an empty
	// session_id and subscribers cannot associate the event with its translation.
	busEvent := events.NewEvent(events.EventType(eventType), message, metadata)
	busEvent.SessionID = sessionID
	s.eventBus.Publish(busEvent)
}

func (s *Server) cleanupRoutine() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		s.cleanupOldSessions()
	}
}

func (s *Server) cleanupOldSessions() {
	s.sessionsMutex.Lock()
	defer s.sessionsMutex.Unlock()

	now := time.Now()
	for sessionID, session := range s.sessions {
		// Remove old completed/failed sessions
		if (session.Status == "completed" || session.Status == "failed" || session.Status == "cancelled") &&
			now.Sub(session.UpdatedAt) > s.config.SessionTimeout {

			// Cancel the per-session timeout context BEFORE dropping the map
			// reference. The context was created via context.WithTimeout in
			// StartTranslation and is never cancelled on the terminal path, so
			// deleting the session without cancelling leaks its timer goroutine
			// until SessionTimeout fires (24h on a real server). cancel() is
			// idempotent — safe even for an already-cancelled session.
			if session.CancelFunc != nil {
				session.CancelFunc()
			}
			delete(s.sessions, sessionID)
			s.logger.Info("Cleaned up old session", map[string]interface{}{
				"session_id": sessionID,
				"status":     session.Status,
				"age":        now.Sub(session.UpdatedAt).String(),
			})
		}
	}
}

// Helper functions

func timeToProto(t time.Time) *timestamppb.Timestamp {
	return timestamppb.New(t)
}

func convertMetadata(metadata map[string]interface{}) map[string]string {
	result := make(map[string]string)
	for k, v := range metadata {
		result[k] = fmt.Sprintf("%v", v)
	}
	return result
}

// NewProviderRegistry creates a new provider registry
func NewProviderRegistry() *ProviderRegistry {
	registry := &ProviderRegistry{
		providers: make(map[string]*proto.ProviderInfo),
	}

	// Initialize with known providers
	registry.initializeDefaultProviders()

	return registry
}

func (pr *ProviderRegistry) initializeDefaultProviders() {
	providers := []*proto.ProviderInfo{
		{
			Name:            "OpenAI GPT",
			Type:            "openai",
			Description:     "OpenAI GPT models (GPT-3.5, GPT-4, etc.)",
			AvailableModels: []string{"gpt-3.5-turbo", "gpt-4", "gpt-4-turbo"},
			Capabilities: map[string]string{
				"languages":    "50+",
				"context_size": "128k",
				"quality":      "high",
			},
			RequiresApiKey:      true,
			RequiresSshConfig:   false,
			RequiresLocalBinary: false,
			Status: &proto.ProviderStatus{
				Available:      true,
				StatusMessage:  "Available",
				ResponseTimeMs: 150,
			},
		},
		{
			Name:            "Anthropic Claude",
			Type:            "anthropic",
			Description:     "Anthropic Claude models (Claude-3, etc.)",
			AvailableModels: []string{"claude-3-opus-20240229", "claude-3-sonnet-20240229", "claude-3-haiku-20240307"},
			Capabilities: map[string]string{
				"languages":    "30+",
				"context_size": "200k",
				"quality":      "very_high",
			},
			RequiresApiKey:      true,
			RequiresSshConfig:   false,
			RequiresLocalBinary: false,
			Status: &proto.ProviderStatus{
				Available:      true,
				StatusMessage:  "Available",
				ResponseTimeMs: 200,
			},
		},
		{
			Name:            "SSH Worker",
			Type:            "ssh",
			Description:     "Remote SSH worker with llama.cpp",
			AvailableModels: []string{"llama2", "mistral", "custom"},
			Capabilities: map[string]string{
				"languages":    "20+",
				"context_size": "4k-32k",
				"quality":      "medium",
				"offline":      "true",
			},
			RequiresApiKey:      false,
			RequiresSshConfig:   true,
			RequiresLocalBinary: false,
			Status: &proto.ProviderStatus{
				Available:      true,
				StatusMessage:  "Available for SSH connections",
				ResponseTimeMs: 100,
			},
		},
	}

	for _, provider := range providers {
		pr.providers[provider.Type] = provider
	}
}

func (pr *ProviderRegistry) GetAll() []*proto.ProviderInfo {
	pr.mutex.RLock()
	defer pr.mutex.RUnlock()

	result := make([]*proto.ProviderInfo, 0, len(pr.providers))
	for _, provider := range pr.providers {
		result = append(result, provider)
	}

	return result
}

func (pr *ProviderRegistry) Get(providerType string) (*proto.ProviderInfo, bool) {
	pr.mutex.RLock()
	defer pr.mutex.RUnlock()

	provider, exists := pr.providers[providerType]
	return provider, exists
}

// GetGRPCServer returns the underlying gRPC server instance
func (s *Server) GetGRPCServer() *grpc.Server {
	return s.grpcServer
}

// Shutdown gracefully shuts down the gRPC server
func (s *Server) Shutdown() {
	s.logger.Info("Shutting down gRPC server", nil)

	// Cancel all active sessions
	s.sessionsMutex.Lock()
	for sessionID, session := range s.sessions {
		if session.CancelFunc != nil {
			session.CancelFunc()
		}
		delete(s.sessions, sessionID)
	}
	s.sessionsMutex.Unlock()

	// Graceful stop gRPC server
	if s.grpcServer != nil {
		s.grpcServer.GracefulStop()
	}

	s.logger.Info("gRPC server shutdown complete", nil)
}
