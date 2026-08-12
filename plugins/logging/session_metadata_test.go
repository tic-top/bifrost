package logging

import (
	"context"
	"testing"

	"github.com/maximhq/bifrost/core/schemas"
	"github.com/stretchr/testify/assert"
)

func TestMergeRealtimeMetadataIncludesGatewaySessionID(t *testing.T) {
	ctx := schemas.NewBifrostContext(context.Background(), schemas.NoDeadline)
	ctx.SetValue(schemas.BifrostContextKeySessionID, "rollout-42")

	got := mergeRealtimeMetadata(map[string]interface{}{"existing": "kept"}, ctx)

	assert.Equal(t, "rollout-42", got["session_id"])
	assert.Equal(t, "kept", got["existing"])
}
