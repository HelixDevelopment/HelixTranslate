package main

// MCP (Model Context Protocol) stdio server for model-bridge.
//
// This is a self-contained implementation of the MCP wire protocol (JSON-RPC 2.0
// over newline-delimited stdin/stdout) — NO external MCP framework dependency is
// added (§11.4.74: a new third-party dependency was deliberately avoided; the
// protocol surface needed here is small and stable). It exposes the strongest
// LLMsVerifier-verified model to a Claude Code agent session as three native MCP
// tools:
//
//	bridge_invoke      {prompt, system?}  → the strongest model's real completion
//	bridge_best_model  {}                 → strongest model metadata + fallback chain
//	bridge_list        {}                 → all verified models, strongest-first
//
// Methods handled: initialize, notifications/initialized, tools/list, tools/call,
// ping. Everything else → JSON-RPC -32601 "method not found".
//
// API-key VALUES are NEVER emitted over the wire (§11.4.10) — only provider IDs,
// model IDs, scores, and real model completions.

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"digital.vasic.translator/pkg/bridge"
)

// flagSetForMCP builds the flag set for the `mcp` subcommand. It uses the same
// bootstrap flags as the other subcommands (commonFlags) so the MCP server reads
// the env keys / db path / verification bounds identically.
func flagSetForMCP() *flag.FlagSet {
	return flag.NewFlagSet("mcp", flag.ContinueOnError)
}

// mcpProtocolVersion is the MCP protocol revision this server advertises.
const mcpProtocolVersion = "2024-11-05"

// jsonRPCError codes (per the JSON-RPC 2.0 spec).
const (
	codeParseError     = -32700
	codeInvalidRequest = -32600
	codeMethodNotFound = -32601
	codeInvalidParams  = -32602
	codeInternalError  = -32603
)

// rpcRequest is an incoming JSON-RPC request (or notification when ID is nil).
type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// rpcResponse is an outgoing JSON-RPC response.
type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// bridgeInvoker is the slice of the bridge the MCP server uses. An interface so
// the server logic is unit-testable with a stub (no real keys / network).
type bridgeInvoker interface {
	Invoke(ctx context.Context, system, prompt string) (string, error)
	ListVerified(ctx context.Context) ([]bridge.ModelInfo, error)
	Source() string
}

// runMCP starts the MCP stdio server. The bridge is opened LAZILY on the first
// tool call that needs it so `model-bridge mcp` can complete the MCP handshake
// (initialize / tools/list) even before any provider key is exercised — the
// agent registers the tools, then the first bridge_invoke materializes the model.
func runMCP(args []string) int {
	fs := flagSetForMCP()
	getOpts := commonFlags(fs)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	opts := getOpts()

	// Lazy opener: the bridge is provisioned (which may run a real verification
	// pass) only when a tool that needs a model is first called.
	var cached *bridge.Bridge
	open := func() (bridgeInvoker, error) {
		if cached != nil {
			return cached, nil
		}
		b, err := openBridge(opts)
		if err != nil {
			return nil, err
		}
		cached = b
		return cached, nil
	}

	srv := &mcpServer{open: open}
	return srv.serve(os.Stdin, os.Stdout)
}

// mcpServer carries the request loop + handlers. open lazily provisions the
// bridge; isolating it behind a func makes the loop fully unit-testable.
type mcpServer struct {
	open func() (bridgeInvoker, error)
}

// serve runs the read→dispatch→write loop over the given streams until EOF.
// Returns 0 on clean EOF, 1 on an unrecoverable I/O error.
func (s *mcpServer) serve(in io.Reader, out io.Writer) int {
	scanner := bufio.NewScanner(in)
	// MCP messages can be large (a full completion echoed back) — raise the line cap.
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	enc := json.NewEncoder(out)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var req rpcRequest
		if err := json.Unmarshal(line, &req); err != nil {
			// Parse error: respond with a null-id error per JSON-RPC.
			_ = enc.Encode(rpcResponse{
				JSONRPC: "2.0",
				Error:   &rpcError{Code: codeParseError, Message: "parse error: " + err.Error()},
			})
			continue
		}

		resp, isNotification := s.dispatch(req)
		if isNotification {
			// Notifications (no id) get no response.
			continue
		}
		if err := enc.Encode(resp); err != nil {
			fmt.Fprintf(os.Stderr, "model-bridge mcp: write response: %v\n", err)
			return 1
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "model-bridge mcp: read stdin: %v\n", err)
		return 1
	}
	return 0
}

// dispatch routes one request to its handler. The second return is true when the
// message is a notification (no id) and therefore gets no response.
func (s *mcpServer) dispatch(req rpcRequest) (rpcResponse, bool) {
	base := rpcResponse{JSONRPC: "2.0", ID: req.ID}
	isNotification := len(req.ID) == 0

	switch req.Method {
	case "initialize":
		base.Result = map[string]any{
			"protocolVersion": mcpProtocolVersion,
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": "model-bridge", "version": "1.0.0"},
		}
		return base, isNotification

	case "notifications/initialized":
		// Client handshake completion — a notification, no response.
		return base, true

	case "ping":
		base.Result = map[string]any{}
		return base, isNotification

	case "tools/list":
		base.Result = map[string]any{"tools": toolDefinitions()}
		return base, isNotification

	case "tools/call":
		result, rerr := s.handleToolCall(req.Params)
		if rerr != nil {
			base.Error = rerr
			return base, isNotification
		}
		base.Result = result
		return base, isNotification

	default:
		if isNotification {
			// Unknown notification: ignore silently per JSON-RPC.
			return base, true
		}
		base.Error = &rpcError{Code: codeMethodNotFound, Message: "method not found: " + req.Method}
		return base, isNotification
	}
}

// toolCallParams is the tools/call params envelope.
type toolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

// handleToolCall executes one of the three bridge tools and returns an MCP
// tool-result object (content array of text blocks). A tool-execution failure is
// reported as an MCP result with isError=true (per the MCP spec) so the agent
// sees the honest error text rather than a transport-level fault.
func (s *mcpServer) handleToolCall(raw json.RawMessage) (map[string]any, *rpcError) {
	var p toolCallParams
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, &rpcError{Code: codeInvalidParams, Message: "invalid tools/call params: " + err.Error()}
	}

	ctx := context.Background()

	switch p.Name {
	case "bridge_invoke":
		var args struct {
			Prompt string `json:"prompt"`
			System string `json:"system"`
		}
		if len(p.Arguments) > 0 {
			if err := json.Unmarshal(p.Arguments, &args); err != nil {
				return nil, &rpcError{Code: codeInvalidParams, Message: "invalid bridge_invoke arguments: " + err.Error()}
			}
		}
		if args.Prompt == "" {
			return toolError("bridge_invoke requires a non-empty 'prompt' argument"), nil
		}
		b, err := s.open()
		if err != nil {
			return toolError(err.Error()), nil
		}
		out, err := b.Invoke(ctx, args.System, args.Prompt)
		if err != nil {
			return toolError(err.Error()), nil
		}
		return toolText(out), nil

	case "bridge_best_model":
		b, err := s.open()
		if err != nil {
			return toolError(err.Error()), nil
		}
		models, err := b.ListVerified(ctx)
		if err != nil {
			return toolError(err.Error()), nil
		}
		payload := map[string]any{
			"source":         b.Source(),
			"strongest":      models[0],
			"fallback_chain": fallbackIDs(models),
		}
		return toolJSON(payload)

	case "bridge_list":
		b, err := s.open()
		if err != nil {
			return toolError(err.Error()), nil
		}
		models, err := b.ListVerified(ctx)
		if err != nil {
			return toolError(err.Error()), nil
		}
		return toolJSON(map[string]any{"source": b.Source(), "models": models})

	default:
		return nil, &rpcError{Code: codeInvalidParams, Message: "unknown tool: " + p.Name}
	}
}

// toolDefinitions returns the MCP tool schema list advertised by tools/list.
func toolDefinitions() []map[string]any {
	return []map[string]any{
		{
			"name":        "bridge_invoke",
			"description": "Send a raw prompt to the strongest LLMsVerifier-verified model and return its real completion. Use for direct model capability access.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"prompt": map[string]any{"type": "string", "description": "The prompt to send to the strongest verified model."},
					"system": map[string]any{"type": "string", "description": "Optional system instruction prepended to the prompt."},
				},
				"required": []string{"prompt"},
			},
		},
		{
			"name":        "bridge_best_model",
			"description": "Return metadata for the strongest verified model plus the deterministic score-descending fallback chain. No API keys are returned.",
			"inputSchema": map[string]any{"type": "object", "properties": map[string]any{}},
		},
		{
			"name":        "bridge_list",
			"description": "List all LLMsVerifier-verified models ranked strongest-first. No API keys are returned.",
			"inputSchema": map[string]any{"type": "object", "properties": map[string]any{}},
		},
	}
}

// toolText wraps a plain string into an MCP tool result.
func toolText(s string) map[string]any {
	return map[string]any{
		"content": []map[string]any{{"type": "text", "text": s}},
	}
}

// toolError wraps an error string into an MCP tool result with isError=true.
func toolError(s string) map[string]any {
	return map[string]any{
		"content": []map[string]any{{"type": "text", "text": "model-bridge: " + s}},
		"isError": true,
	}
}

// toolJSON marshals v indented and wraps it as an MCP text result.
func toolJSON(v any) (map[string]any, *rpcError) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, &rpcError{Code: codeInternalError, Message: "marshal tool result: " + err.Error()}
	}
	return toolText(string(b)), nil
}
