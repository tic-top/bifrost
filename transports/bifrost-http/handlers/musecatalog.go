package handlers

import "github.com/maximhq/bifrost/core/schemas"

const (
	defaultMuseContextWindow   = 262144
	defaultMuseMaxOutputTokens = 32768
)

type museCatalogResponse struct {
	Data []museCatalogModel `json:"data"`
}

type museCatalogModel struct {
	ID              string   `json:"id"`
	Object          string   `json:"object"`
	Created         int64    `json:"created"`
	OwnedBy         string   `json:"owned_by"`
	Name            string   `json:"name"`
	DisplayName     string   `json:"display_name"`
	ContextWindow   int      `json:"context_window"`
	MaxOutputTokens int      `json:"max_output_tokens"`
	Capabilities    []string `json:"capabilities"`
}

// projectMuseCatalog maps Bifrost's standard model list onto the catalog shape
// consumed by Muse Code 0.1.x for custom --base-url endpoints. Defaults match
// the compatibility contract used for bare OpenAI/vLLM catalogs; richer model
// metadata wins whenever Bifrost has it.
func projectMuseCatalog(resp *schemas.BifrostListModelsResponse) museCatalogResponse {
	result := museCatalogResponse{Data: make([]museCatalogModel, 0)}
	if resp == nil {
		return result
	}

	result.Data = make([]museCatalogModel, 0, len(resp.Data))
	for _, model := range resp.Data {
		if model.ID == "" {
			continue
		}

		entry := museCatalogModel{
			ID:              model.ID,
			Object:          "model",
			OwnedBy:         "bifrost",
			Name:            model.ID,
			DisplayName:     model.ID,
			ContextWindow:   defaultMuseContextWindow,
			MaxOutputTokens: defaultMuseMaxOutputTokens,
			Capabilities:    []string{"tools"},
		}
		if model.Created != nil {
			entry.Created = *model.Created
		}
		if model.OwnedBy != nil && *model.OwnedBy != "" {
			entry.OwnedBy = *model.OwnedBy
		}
		if model.ContextLength != nil && *model.ContextLength > 0 {
			entry.ContextWindow = *model.ContextLength
		} else if model.TopProvider != nil && model.TopProvider.ContextLength != nil && *model.TopProvider.ContextLength > 0 {
			entry.ContextWindow = *model.TopProvider.ContextLength
		}
		if model.MaxOutputTokens != nil && *model.MaxOutputTokens > 0 {
			entry.MaxOutputTokens = *model.MaxOutputTokens
		} else if model.TopProvider != nil && model.TopProvider.MaxCompletionTokens != nil && *model.TopProvider.MaxCompletionTokens > 0 {
			entry.MaxOutputTokens = *model.TopProvider.MaxCompletionTokens
		}
		result.Data = append(result.Data, entry)
	}
	return result
}
