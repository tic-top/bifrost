package logging

import (
	"context"
	"testing"
	"time"

	"github.com/maximhq/bifrost/core/schemas"
	"github.com/maximhq/bifrost/framework/logstore"
)

func TestHTTPTransportHooksRecordClientWireOnPendingLog(t *testing.T) {
	plugin := &LoggerPlugin{logger: testLogger{}}
	ctx := schemas.NewBifrostContext(context.Background(), schemas.NoDeadline)
	ctx.SetValue(schemas.BifrostContextKeyTraceID, "client-wire-trace")
	ctx.SetValue(schemas.BifrostContextKeyShouldStoreRawInLogs, true)

	entry := &logstore.Log{ID: "client-wire-request"}
	plugin.pendingLogsToInject.Store("client-wire-trace", &pendingInjectEntries{
		entries:   []*logstore.Log{entry},
		createdAt: time.Now(),
	})

	req := &schemas.HTTPRequest{
		Method: "POST",
		Path:   "/genai/v1beta/models/vllm/Qwen3.6-27B:streamGenerateContent",
		Body:   []byte(`{"contents":[{"parts":[{"text":"hello"}]}]}`),
	}
	resp := &schemas.HTTPResponse{StatusCode: 200, Body: []byte("data: {\"candidates\":[]}\n\n")}
	if _, err := plugin.HTTPTransportPreHook(ctx, req); err != nil {
		t.Fatalf("HTTPTransportPreHook() error = %v", err)
	}
	if err := plugin.HTTPTransportPostHook(ctx, req, resp); err != nil {
		t.Fatalf("HTTPTransportPostHook() error = %v", err)
	}

	if entry.PassthroughRequestBody != string(req.Body) {
		t.Fatalf("client request wire not recorded: got %q", entry.PassthroughRequestBody)
	}
	if entry.PassthroughResponseBody != string(resp.Body) {
		t.Fatalf("client response wire not recorded: got %q", entry.PassthroughResponseBody)
	}
	params, ok := entry.ParamsParsed.(map[string]interface{})
	if !ok || params["path"] != req.Path || params["method"] != "POST" || params["status_code"] != 200 {
		t.Fatalf("client wire metadata not recorded: %#v", entry.ParamsParsed)
	}
}

func TestHTTPTransportHooksDoNotRecordClientWireWhenRawStorageIsDisabled(t *testing.T) {
	plugin := &LoggerPlugin{logger: testLogger{}}
	ctx := schemas.NewBifrostContext(context.Background(), schemas.NoDeadline)
	ctx.SetValue(schemas.BifrostContextKeyTraceID, "client-wire-disabled")
	ctx.SetValue(schemas.BifrostContextKeyShouldStoreRawInLogs, false)
	entry := &logstore.Log{ID: "client-wire-disabled-request"}
	plugin.pendingLogsToInject.Store("client-wire-disabled", &pendingInjectEntries{
		entries:   []*logstore.Log{entry},
		createdAt: time.Now(),
	})

	req := &schemas.HTTPRequest{Body: []byte(`{"contents":[]}`)}
	resp := &schemas.HTTPResponse{Body: []byte("data: {}\n\n")}
	if _, err := plugin.HTTPTransportPreHook(ctx, req); err != nil {
		t.Fatalf("HTTPTransportPreHook() error = %v", err)
	}
	if err := plugin.HTTPTransportPostHook(ctx, req, resp); err != nil {
		t.Fatalf("HTTPTransportPostHook() error = %v", err)
	}
	if entry.PassthroughRequestBody != "" || entry.PassthroughResponseBody != "" {
		t.Fatalf("client wire recorded while disabled: request=%q response=%q", entry.PassthroughRequestBody, entry.PassthroughResponseBody)
	}
}
