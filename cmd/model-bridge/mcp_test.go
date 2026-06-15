package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"digital.vasic.translator/pkg/bridge"
)

// stubInvoker is an in-memory bridgeInvoker for protocol-level MCP tests. It
// returns a deterministic, UNFORGEABLE token from Invoke so the §11.4.78-style
// challenge can assert the MCP path actually carried the model's response through
// the tool-call plumbing (a server that returned canned text would not echo the
// per-call nonce).
type stubInvoker struct {
	invokeFn func(ctx context.Context, system, prompt string) (string, error)
	models   []bridge.ModelInfo
	source   string
	listErr  error
}

func (s *stubInvoker) Invoke(ctx context.Context, system, prompt string) (string, error) {
	return s.invokeFn(ctx, system, prompt)
}

func (s *stubInvoker) ListVerified(ctx context.Context) ([]bridge.ModelInfo, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	return s.models, nil
}

func (s *stubInvoker) Source() string { return s.source }

// newTestServer builds an mcpServer whose lazy opener returns the stub.
func newTestServer(stub *stubInvoker) *mcpServer {
	return &mcpServer{open: func() (bridgeInvoker, error) { return stub, nil }}
}

// call drives one request through dispatch and returns the response.
func call(t *testing.T, s *mcpServer, method string, id any, params any) rpcResponse {
	t.Helper()
	var rawID json.RawMessage
	if id != nil {
		b, _ := json.Marshal(id)
		rawID = b
	}
	var rawParams json.RawMessage
	if params != nil {
		b, _ := json.Marshal(params)
		rawParams = b
	}
	resp, _ := s.dispatch(rpcRequest{JSONRPC: "2.0", ID: rawID, Method: method, Params: rawParams})
	return resp
}

func TestMCP_Initialize(t *testing.T) {
	s := newTestServer(&stubInvoker{})
	resp := call(t, s, "initialize", 1, nil)
	if resp.Error != nil {
		t.Fatalf("initialize error: %+v", resp.Error)
	}
	res, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatalf("initialize result not an object: %T", resp.Result)
	}
	if res["protocolVersion"] != mcpProtocolVersion {
		t.Errorf("protocolVersion = %v, want %s", res["protocolVersion"], mcpProtocolVersion)
	}
	caps, _ := res["capabilities"].(map[string]any)
	if _, ok := caps["tools"]; !ok {
		t.Errorf("initialize must advertise tools capability, got %v", caps)
	}
}

func TestMCP_ToolsList_AdvertisesAllThree(t *testing.T) {
	s := newTestServer(&stubInvoker{})
	resp := call(t, s, "tools/list", 2, nil)
	if resp.Error != nil {
		t.Fatalf("tools/list error: %+v", resp.Error)
	}
	res := resp.Result.(map[string]any)
	tools := res["tools"].([]map[string]any)
	names := map[string]bool{}
	for _, tl := range tools {
		names[tl["name"].(string)] = true
	}
	for _, want := range []string{"bridge_invoke", "bridge_best_model", "bridge_list"} {
		if !names[want] {
			t.Errorf("tools/list missing %q (got %v)", want, names)
		}
	}
}

// TestMCP_ToolsCall_Invoke_UnforgeableChallenge is the §11.4.78-style unforgeable
// challenge: the prompt embeds a nonce the stub model incorporates into its
// answer. A bridge_invoke that genuinely routed the prompt to the model and
// returned its completion yields the nonce; a canned/short-circuited response
// would not. This proves the MCP tool-call plumbing carries the real model
// round-trip end to end.
func TestMCP_ToolsCall_Invoke_UnforgeableChallenge(t *testing.T) {
	const nonce = "ATM-NONCE-7f3c91"
	var gotPrompt, gotSystem string
	stub := &stubInvoker{
		invokeFn: func(_ context.Context, system, prompt string) (string, error) {
			gotPrompt, gotSystem = prompt, system
			// A real model would answer using the prompt; echo the nonce back as
			// the verifiable signal it actually processed THIS prompt.
			return "model-says: " + nonce, nil
		},
	}
	s := newTestServer(stub)

	resp := call(t, s, "tools/call", 3, toolCallParams{
		Name:      "bridge_invoke",
		Arguments: json.RawMessage(fmt.Sprintf(`{"prompt":"echo the token %s","system":"be terse"}`, nonce)),
	})
	if resp.Error != nil {
		t.Fatalf("tools/call invoke transport error: %+v", resp.Error)
	}
	res := resp.Result.(map[string]any)
	if res["isError"] == true {
		t.Fatalf("invoke reported tool error: %v", res["content"])
	}
	content := res["content"].([]map[string]any)
	text := content[0]["text"].(string)
	if !strings.Contains(text, nonce) {
		t.Errorf("invoke result %q does not carry the unforgeable nonce %q — the model round-trip was not exercised", text, nonce)
	}
	if !strings.Contains(gotPrompt, nonce) {
		t.Errorf("the prompt reaching the model %q did not carry the nonce", gotPrompt)
	}
	if gotSystem != "be terse" {
		t.Errorf("system instruction not threaded to the model, got %q", gotSystem)
	}
}

func TestMCP_ToolsCall_Invoke_EmptyPromptIsToolError(t *testing.T) {
	s := newTestServer(&stubInvoker{invokeFn: func(context.Context, string, string) (string, error) {
		t.Fatal("Invoke must not be called with an empty prompt")
		return "", nil
	}})
	resp := call(t, s, "tools/call", 4, toolCallParams{Name: "bridge_invoke", Arguments: json.RawMessage(`{"prompt":""}`)})
	res := resp.Result.(map[string]any)
	if res["isError"] != true {
		t.Errorf("empty prompt must yield isError=true, got %v", res)
	}
}

// TestMCP_ToolsCall_Invoke_ModelErrorIsHonest proves a model/bridge failure is
// surfaced as an honest MCP tool error (isError=true), never a faked success.
func TestMCP_ToolsCall_Invoke_ModelErrorIsHonest(t *testing.T) {
	s := newTestServer(&stubInvoker{invokeFn: func(context.Context, string, string) (string, error) {
		return "", fmt.Errorf("provider quota exceeded")
	}})
	resp := call(t, s, "tools/call", 5, toolCallParams{Name: "bridge_invoke", Arguments: json.RawMessage(`{"prompt":"hi"}`)})
	res := resp.Result.(map[string]any)
	if res["isError"] != true {
		t.Fatalf("model error must yield isError=true, got %v", res)
	}
	text := res["content"].([]map[string]any)[0]["text"].(string)
	if !strings.Contains(text, "quota exceeded") {
		t.Errorf("tool error text should carry the real error, got %q", text)
	}
}

func TestMCP_ToolsCall_BestModel(t *testing.T) {
	s := newTestServer(&stubInvoker{
		source: "in-process",
		models: []bridge.ModelInfo{
			{ProviderID: "openai", ModelID: "gpt-4o", Score: 0.9, FallbackOrder: 1, FactoryName: "openai"},
			{ProviderID: "deepseek", ModelID: "deepseek-chat", Score: 0.7, FallbackOrder: 2, FactoryName: "openai"},
		},
	})
	resp := call(t, s, "tools/call", 6, toolCallParams{Name: "bridge_best_model"})
	res := resp.Result.(map[string]any)
	if res["isError"] == true {
		t.Fatalf("best_model reported error: %v", res)
	}
	var payload struct {
		Source    string `json:"source"`
		Strongest struct {
			ModelID string `json:"model_id"`
		} `json:"strongest"`
		FallbackChain []string `json:"fallback_chain"`
	}
	text := res["content"].([]map[string]any)[0]["text"].(string)
	if err := json.Unmarshal([]byte(text), &payload); err != nil {
		t.Fatalf("best_model result not valid JSON: %v\n%s", err, text)
	}
	if payload.Strongest.ModelID != "gpt-4o" {
		t.Errorf("strongest = %q, want gpt-4o", payload.Strongest.ModelID)
	}
	if len(payload.FallbackChain) != 2 || payload.FallbackChain[0] != "openai/gpt-4o" {
		t.Errorf("fallback chain = %v, want [openai/gpt-4o deepseek/deepseek-chat]", payload.FallbackChain)
	}
}

func TestMCP_ToolsCall_UnknownTool(t *testing.T) {
	s := newTestServer(&stubInvoker{})
	resp := call(t, s, "tools/call", 7, toolCallParams{Name: "no_such_tool"})
	if resp.Error == nil || resp.Error.Code != codeInvalidParams {
		t.Errorf("unknown tool should yield invalid-params error, got %+v", resp.Error)
	}
}

func TestMCP_UnknownMethod(t *testing.T) {
	s := newTestServer(&stubInvoker{})
	resp := call(t, s, "frobnicate", 8, nil)
	if resp.Error == nil || resp.Error.Code != codeMethodNotFound {
		t.Errorf("unknown method should yield method-not-found, got %+v", resp.Error)
	}
}

func TestMCP_Notification_GetsNoResponse(t *testing.T) {
	s := newTestServer(&stubInvoker{})
	_, isNotification := s.dispatch(rpcRequest{JSONRPC: "2.0", Method: "notifications/initialized"})
	if !isNotification {
		t.Error("notifications/initialized must be treated as a notification (no response)")
	}
}

// TestMCP_Serve_EndToEnd drives the full read→dispatch→write loop over real
// byte streams, proving newline-delimited JSON-RPC framing works.
func TestMCP_Serve_EndToEnd(t *testing.T) {
	const nonce = "E2E-NONCE-44ab"
	stub := &stubInvoker{invokeFn: func(_ context.Context, _, prompt string) (string, error) {
		return "echo:" + nonce, nil
	}}
	s := newTestServer(stub)

	in := strings.NewReader(strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize"}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		fmt.Sprintf(`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"bridge_invoke","arguments":{"prompt":"x %s"}}}`, nonce),
	}, "\n") + "\n")

	var out strings.Builder
	if rc := s.serve(in, &out); rc != 0 {
		t.Fatalf("serve returned %d, want 0", rc)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	// 3 responses (initialize, tools/list, tools/call) — the notification yields none.
	if len(lines) != 3 {
		t.Fatalf("got %d response lines, want 3:\n%s", len(lines), out.String())
	}
	if !strings.Contains(lines[2], nonce) {
		t.Errorf("final tools/call response did not carry the nonce: %s", lines[2])
	}
}
