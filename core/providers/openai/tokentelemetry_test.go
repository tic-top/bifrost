package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/maximhq/bifrost/core/schemas"
)

func TestPassthroughTokenTelemetryIsDoubleOptInAndPreservesClientBody(t *testing.T) {
	tests := []struct {
		name      string
		allowed   bool
		requested bool
		wantLog   bool
		wantIDs   bool
	}{
		{name: "provider and request opt in", allowed: true, requested: true, wantLog: true, wantIDs: true},
		{name: "provider allows but request does not opt in", allowed: true},
		{name: "request opts in but provider denies", requested: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seen := make(chan map[string]interface{}, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body map[string]interface{}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Errorf("decode upstream request: %v", err)
				}
				seen <- body
				w.Header().Set("Content-Type", "application/json")
				_, _ = fmt.Fprint(w, `{"id":"chatcmpl_1","object":"chat.completion","model":"test","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`)
			}))
			defer server.Close()

			provider := NewOpenAIProvider(&schemas.ProviderConfig{
				NetworkConfig: schemas.NetworkConfig{BaseURL: server.URL},
				OpenAIConfig: &schemas.OpenAIConfig{
					AllowTokenTelemetry: tt.allowed,
				},
			}, passthroughTestLogger{})
			ctx := schemas.NewBifrostContext(context.Background(), schemas.NoDeadline)
			if tt.requested {
				ctx.SetValue(schemas.BifrostContextKeyTokenTelemetryRequested, true)
			}
			original := []byte(`{"model":"test","messages":[{"role":"user","content":"hi"}],"stream":false}`)
			req := &schemas.BifrostPassthroughRequest{
				Method: http.MethodPost,
				Path:   "/v1/chat/completions",
				Body:   append([]byte(nil), original...),
				SafeHeaders: map[string]string{
					"content-type": "application/json",
				},
				Provider: schemas.OpenAI,
				Model:    "test",
			}

			if _, err := provider.Passthrough(ctx, schemas.Key{}, req); err != nil {
				t.Fatalf("Passthrough returned error: %v", err)
			}
			upstream := <-seen
			if got, ok := upstream["logprobs"].(bool); ok != tt.wantLog || got != tt.wantLog {
				t.Fatalf("upstream logprobs = %#v, want %v", upstream["logprobs"], tt.wantLog)
			}
			if got, ok := upstream["return_token_ids"].(bool); ok != tt.wantIDs || got != tt.wantIDs {
				t.Fatalf("upstream return_token_ids = %#v, want %v", upstream["return_token_ids"], tt.wantIDs)
			}
			if string(req.Body) != string(original) {
				t.Fatalf("client request body mutated: got %s want %s", req.Body, original)
			}
		})
	}
}

func TestPassthroughTokenTelemetryPreservesExplicitClientValues(t *testing.T) {
	seen := make(chan map[string]interface{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		seen <- body
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"choices":[]}`)
	}))
	defer server.Close()

	provider := NewOpenAIProvider(&schemas.ProviderConfig{
		NetworkConfig: schemas.NetworkConfig{BaseURL: server.URL},
		OpenAIConfig:  &schemas.OpenAIConfig{AllowTokenTelemetry: true},
	}, passthroughTestLogger{})
	ctx := schemas.NewBifrostContext(context.Background(), schemas.NoDeadline)
	ctx.SetValue(schemas.BifrostContextKeyTokenTelemetryRequested, true)
	req := &schemas.BifrostPassthroughRequest{
		Method:      http.MethodPost,
		Path:        "/v1/chat/completions",
		Body:        []byte(`{"model":"test","messages":[],"logprobs":false,"return_token_ids":false}`),
		SafeHeaders: map[string]string{"content-type": "application/json"},
	}

	if _, err := provider.Passthrough(ctx, schemas.Key{}, req); err != nil {
		t.Fatalf("Passthrough returned error: %v", err)
	}
	upstream := <-seen
	if upstream["logprobs"] != false || upstream["return_token_ids"] != false {
		t.Fatalf("explicit client values overwritten: %#v", upstream)
	}
}

func TestNormalizedChatTokenTelemetryInjectsWithoutMutatingRequest(t *testing.T) {
	seen := make(chan map[string]interface{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		seen <- body
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"id":"chat-1","object":"chat.completion","created":1,"model":"test","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop","token_ids":[7],"logprobs":{"content":[{"token":"ok","logprob":-0.1}]}}],"prompt_token_ids":[1,2,3],"usage":{"prompt_tokens":3,"completion_tokens":1,"total_tokens":4}}`)
	}))
	defer server.Close()

	provider := NewOpenAIProvider(&schemas.ProviderConfig{
		NetworkConfig: schemas.NetworkConfig{BaseURL: server.URL},
		OpenAIConfig:  &schemas.OpenAIConfig{AllowTokenTelemetry: true},
	}, passthroughTestLogger{})
	ctx := schemas.NewBifrostContext(context.Background(), schemas.NoDeadline)
	ctx.SetValue(schemas.BifrostContextKeyTokenTelemetryRequested, true)
	request := &schemas.BifrostChatRequest{
		Provider: schemas.OpenAI,
		Model:    "test",
		Input: []schemas.ChatMessage{{
			Role:    schemas.ChatMessageRoleUser,
			Content: &schemas.ChatMessageContent{ContentStr: schemas.Ptr("hi")},
		}},
		Params: &schemas.ChatParameters{},
	}

	if _, err := provider.ChatCompletion(ctx, schemas.Key{}, request); err != nil {
		t.Fatalf("ChatCompletion returned error: %v", err)
	}
	upstream := <-seen
	if upstream["logprobs"] != true || upstream["return_token_ids"] != true {
		t.Fatalf("normalized upstream request lacks telemetry params: %#v", upstream)
	}
	if request.Params.LogProbs != nil || len(request.Params.ExtraParams) != 0 {
		t.Fatalf("normalized client request mutated: %#v", request.Params)
	}
}

func TestNormalizedChatTokenTelemetryPreservesExplicitFalse(t *testing.T) {
	provider := &OpenAIProvider{allowTokenTelemetry: true}
	ctx := schemas.NewBifrostContext(context.Background(), schemas.NoDeadline)
	ctx.SetValue(schemas.BifrostContextKeyTokenTelemetryRequested, true)
	request := &schemas.BifrostChatRequest{Params: &schemas.ChatParameters{
		LogProbs:    schemas.Ptr(false),
		ExtraParams: map[string]interface{}{"return_token_ids": false},
	}}

	prepared := provider.prepareChatTokenTelemetry(ctx, request)
	if prepared.Params.LogProbs == nil || *prepared.Params.LogProbs || prepared.Params.ExtraParams["return_token_ids"] != false {
		t.Fatalf("explicit false values overwritten: %#v", prepared.Params)
	}
}

func TestNormalizedChatStreamRawTelemetryKeepsPromptOnlyChunk(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w,
			"data: {\"id\":\"chat-1\",\"object\":\"chat.completion.chunk\",\"created\":1,\"model\":\"test\",\"prompt_token_ids\":[1,2,3],\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"\"}}]}\n\n"+
				"data: {\"id\":\"chat-1\",\"object\":\"chat.completion.chunk\",\"created\":1,\"model\":\"test\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"ok\"},\"token_ids\":[7],\"logprobs\":{\"content\":[{\"token\":\"ok\",\"logprob\":-0.1}]}}]}\n\n"+
				"data: {\"id\":\"chat-1\",\"object\":\"chat.completion.chunk\",\"created\":1,\"model\":\"test\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n"+
				"data: [DONE]\n\n")
	}))
	defer server.Close()

	provider := NewOpenAIProvider(&schemas.ProviderConfig{
		NetworkConfig:           schemas.NetworkConfig{BaseURL: server.URL},
		StoreRawRequestResponse: true,
		OpenAIConfig:            &schemas.OpenAIConfig{AllowTokenTelemetry: true},
	}, passthroughTestLogger{})
	ctx := schemas.NewBifrostContext(context.Background(), schemas.NoDeadline)
	ctx.SetValue(schemas.BifrostContextKeyTokenTelemetryRequested, true)
	ctx.SetValue(schemas.BifrostContextKeyCaptureRawRequest, true)
	ctx.SetValue(schemas.BifrostContextKeyCaptureRawResponse, true)
	request := &schemas.BifrostChatRequest{
		Provider: schemas.OpenAI,
		Model:    "test",
		Input: []schemas.ChatMessage{{
			Role:    schemas.ChatMessageRoleUser,
			Content: &schemas.ChatMessageContent{ContentStr: schemas.Ptr("hi")},
		}},
		Params: &schemas.ChatParameters{},
	}

	postHook := func(_ *schemas.BifrostContext, result *schemas.BifrostResponse, err *schemas.BifrostError) (*schemas.BifrostResponse, *schemas.BifrostError) {
		return result, err
	}
	stream, err := provider.ChatCompletionStream(ctx, postHook, nil, schemas.Key{}, request)
	if err != nil {
		t.Fatalf("ChatCompletionStream returned error: %v", err)
	}
	var raw strings.Builder
	for chunk := range stream {
		if chunk.BifrostError != nil {
			t.Fatalf("stream error: %v", chunk.BifrostError)
		}
		if chunk.BifrostChatResponse != nil && chunk.BifrostChatResponse.ExtraFields.RawResponse != nil {
			raw.WriteString(fmt.Sprintf("%v", chunk.BifrostChatResponse.ExtraFields.RawResponse))
		}
	}
	if !strings.Contains(raw.String(), `"prompt_token_ids":[1,2,3]`) {
		t.Fatalf("raw stream dropped provider-only prompt token chunk: %s", raw.String())
	}
}
