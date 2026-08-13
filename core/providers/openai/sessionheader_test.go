package openai

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maximhq/bifrost/core/schemas"
)

func TestResponsesForwardsConfiguredUpstreamSessionHeader(t *testing.T) {
	const sessionID = "cache-locality-task-42"
	seen := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen <- r.Header.Get("x-genai-session-id")
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"id":"resp_1","object":"response","created_at":1,"status":"completed","model":"gpt-test","output":[{"id":"msg_1","type":"message","status":"completed","role":"assistant","content":[{"type":"output_text","text":"ok","annotations":[],"logprobs":[]}]}],"usage":{"input_tokens":1,"input_tokens_details":{"cached_tokens":0},"output_tokens":1,"total_tokens":2}}`)
	}))
	defer server.Close()

	provider := NewOpenAIProvider(&schemas.ProviderConfig{
		NetworkConfig: schemas.NetworkConfig{BaseURL: server.URL},
		OpenAIConfig: &schemas.OpenAIConfig{
			UpstreamSessionHeader: "x-genai-session-id",
		},
	}, passthroughTestLogger{})
	ctx := schemas.NewBifrostContext(context.Background(), schemas.NoDeadline)
	ctx.SetValue(schemas.BifrostContextKeySessionID, sessionID)
	key := schemas.Key{Value: *schemas.NewSecretVar("test-key")}
	request := &schemas.BifrostResponsesRequest{
		Provider: schemas.OpenAI,
		Model:    "gpt-test",
		Input: []schemas.ResponsesMessage{{
			Type:    schemas.Ptr(schemas.ResponsesMessageTypeMessage),
			Role:    schemas.Ptr(schemas.ResponsesInputMessageRoleUser),
			Content: &schemas.ResponsesMessageContent{ContentStr: schemas.Ptr("hello")},
		}},
	}

	if _, err := provider.Responses(ctx, key, request); err != nil {
		t.Fatalf("Responses returned error: %v", err)
	}
	if got := <-seen; got != sessionID {
		t.Fatalf("upstream session header = %q, want %q", got, sessionID)
	}
}
