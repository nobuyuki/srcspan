package diagnostic

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nobuyuki/srcspan/files"
)

// The outputs of serde_json with codespan-reporting 0.13.1 (the
// `serialization` feature), for the same diagnostics.
const (
	rustDiagnosticJSON = `{"severity":"Error","code":"E0001","message":"expected ` + "`;`" + `",` +
		`"labels":[{"style":"Primary","file_id":0,"range":{"start":9,"end":9},"message":"here"},` +
		`{"style":"Secondary","file_id":1,"range":{"start":2,"end":5},"message":""}],"notes":["a note"]}`
	rustNoCodeJSON = `{"severity":"Warning","code":null,"message":"","labels":[],"notes":[]}`
)

func testDiagnostic() *Diagnostic {
	return NewError().
		WithMessage("expected `;`").
		WithCode("E0001").
		WithLabels([]Label{
			NewPrimary(0, files.NewRange(9, 9)).WithMessage("here"),
			NewSecondary(1, files.NewRange(2, 5)),
		}).
		WithNotes([]string{"a note"})
}

func TestMarshalLikeRust(t *testing.T) {
	data, err := json.Marshal(testDiagnostic())
	require.NoError(t, err)
	assert.Equal(t, rustDiagnosticJSON, string(data)) //nolint:testifylint // byte-for-byte equal to serde_json, including field order

	// No code is `null`, and no labels and notes are `[]`.
	data, err = json.Marshal(NewWarning())
	require.NoError(t, err)
	assert.Equal(t, rustNoCodeJSON, string(data)) //nolint:testifylint // byte-for-byte equal to serde_json, including field order

	data, err = json.Marshal(Diagnostic{Severity: Warning})
	require.NoError(t, err)
	assert.Equal(t, rustNoCodeJSON, string(data)) //nolint:testifylint // byte-for-byte equal to serde_json, including field order
}

func TestUnmarshalFromRust(t *testing.T) {
	var d Diagnostic
	require.NoError(t, json.Unmarshal([]byte(rustDiagnosticJSON), &d))
	assert.Equal(t, *testDiagnostic(), d)

	require.NoError(t, json.Unmarshal([]byte(rustNoCodeJSON), &d))
	assert.Equal(t, *NewWarning(), d)

	// A missing code is `None` in Rust too.
	require.NoError(t, json.Unmarshal([]byte(`{"severity":"Bug","message":"m","labels":[],"notes":[]}`), &d))
	assert.Equal(t, *NewBug().WithMessage("m"), d)
}

func TestSeverityAndLabelStyleNames(t *testing.T) {
	// The same as serde_json in Rust.
	for severity, expected := range map[Severity]string{
		Bug: `"Bug"`, Error: `"Error"`, Warning: `"Warning"`, Note: `"Note"`, Help: `"Help"`,
	} {
		data, err := json.Marshal(severity)
		require.NoError(t, err)
		assert.Equal(t, expected, string(data))

		var decoded Severity
		require.NoError(t, json.Unmarshal(data, &decoded))
		assert.Equal(t, severity, decoded)
	}
	for style, expected := range map[LabelStyle]string{Primary: `"Primary"`, Secondary: `"Secondary"`} {
		data, err := json.Marshal(style)
		require.NoError(t, err)
		assert.Equal(t, expected, string(data))

		var decoded LabelStyle
		require.NoError(t, json.Unmarshal(data, &decoded))
		assert.Equal(t, style, decoded)
	}

	// Names are case-sensitive, like serde.
	var severity Severity
	require.ErrorContains(t, json.Unmarshal([]byte(`"error"`), &severity),
		"unknown variant `error`, expected one of `Help`, `Note`, `Warning`, `Error`, `Bug`")
}
