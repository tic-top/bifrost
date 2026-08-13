package handlers

import (
	"reflect"
	"testing"

	"github.com/fasthttp/router"
	"github.com/maximhq/bifrost/core/schemas"
	"github.com/valyala/fasthttp"
)

func TestCompletionHandlerRegistersMuseCatalogRoutes(t *testing.T) {
	r := router.New()
	h := &CompletionHandler{}
	shortCircuit := func(next fasthttp.RequestHandler) fasthttp.RequestHandler {
		return func(ctx *fasthttp.RequestCtx) {
			ctx.SetStatusCode(fasthttp.StatusNoContent)
		}
	}
	h.RegisterRoutes(r, schemas.BifrostHTTPMiddleware(shortCircuit))

	for _, path := range []string{"/muse-code/models", "/v1/muse-code/models"} {
		t.Run(path, func(t *testing.T) {
			var ctx fasthttp.RequestCtx
			ctx.Request.Header.SetMethod(fasthttp.MethodGet)
			ctx.Request.SetRequestURI(path)

			r.Handler(&ctx)

			if got := ctx.Response.StatusCode(); got != fasthttp.StatusNoContent {
				t.Fatalf("GET %s returned %d, want %d", path, got, fasthttp.StatusNoContent)
			}
		})
	}
}

func TestProjectMuseCatalog(t *testing.T) {
	created := int64(123)
	ownedBy := "vllm"
	contextLength := 131072
	maxOutputTokens := 8192
	topContextLength := 65536
	topMaxOutputTokens := 4096

	got := projectMuseCatalog(&schemas.BifrostListModelsResponse{Data: []schemas.Model{
		{
			ID:              "vllm/qwen",
			Created:         &created,
			OwnedBy:         &ownedBy,
			ContextLength:   &contextLength,
			MaxOutputTokens: &maxOutputTokens,
		},
		{
			ID: "xai/grok",
			TopProvider: &schemas.TopProvider{
				ContextLength:       &topContextLength,
				MaxCompletionTokens: &topMaxOutputTokens,
			},
		},
		{ID: ""},
	}})

	want := museCatalogResponse{Data: []museCatalogModel{
		{
			ID: "vllm/qwen", Object: "model", Created: 123, OwnedBy: "vllm",
			Name: "vllm/qwen", DisplayName: "vllm/qwen", ContextWindow: 131072,
			MaxOutputTokens: 8192, Capabilities: []string{"tools"},
		},
		{
			ID: "xai/grok", Object: "model", OwnedBy: "bifrost",
			Name: "xai/grok", DisplayName: "xai/grok", ContextWindow: 65536,
			MaxOutputTokens: 4096, Capabilities: []string{"tools"},
		},
	}}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("projectMuseCatalog() = %#v, want %#v", got, want)
	}
}

func TestProjectMuseCatalogUsesSafeDefaults(t *testing.T) {
	got := projectMuseCatalog(&schemas.BifrostListModelsResponse{Data: []schemas.Model{{ID: "custom/model"}}})
	if len(got.Data) != 1 {
		t.Fatalf("got %d models, want 1", len(got.Data))
	}
	model := got.Data[0]
	if model.ContextWindow != defaultMuseContextWindow || model.MaxOutputTokens != defaultMuseMaxOutputTokens {
		t.Fatalf("unexpected defaults: context=%d output=%d", model.ContextWindow, model.MaxOutputTokens)
	}
}
