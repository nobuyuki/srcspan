package srcspan

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The same as serde_json with codespan 0.13.1 (the `serialization` feature).
func TestSpanAndLocationJSONLikeRust(t *testing.T) {
	data, err := json.Marshal(NewSpan(0, 4))
	require.NoError(t, err)
	assert.Equal(t, `{"start":0,"end":4}`, string(data))

	var span Span
	require.NoError(t, json.Unmarshal(data, &span))
	assert.Equal(t, NewSpan(0, 4), span)

	data, err = json.Marshal(NewLocation(1, 3))
	require.NoError(t, err)
	assert.Equal(t, `{"line":1,"column":3}`, string(data))

	var location Location
	require.NoError(t, json.Unmarshal(data, &location))
	assert.Equal(t, NewLocation(1, 3), location)
}
