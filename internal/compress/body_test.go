package compress

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompressDecompress_RoundTrip(t *testing.T) {
	text := "Hello, this is the plain text body."
	html := "<html><body><h1>Hello</h1><p>This is the HTML body.</p></body></html>"

	compressed, err := CompressBody(text, html)
	require.NoError(t, err)
	require.NotEmpty(t, compressed)

	gotText, gotHTML, err := DecompressBody(compressed)
	require.NoError(t, err)
	assert.Equal(t, text, gotText)
	assert.Equal(t, html, gotHTML)
}

func TestCompressBody_EmptyBodies(t *testing.T) {
	compressed, err := CompressBody("", "")
	require.NoError(t, err)
	assert.Nil(t, compressed)
}

func TestDecompressBody_Nil(t *testing.T) {
	text, html, err := DecompressBody(nil)
	require.NoError(t, err)
	assert.Empty(t, text)
	assert.Empty(t, html)
}

func TestCompressBody_OnlyText(t *testing.T) {
	compressed, err := CompressBody("text only", "")
	require.NoError(t, err)
	require.NotEmpty(t, compressed)

	text, html, err := DecompressBody(compressed)
	require.NoError(t, err)
	assert.Equal(t, "text only", text)
	assert.Empty(t, html)
}

func TestCompressBody_OnlyHTML(t *testing.T) {
	compressed, err := CompressBody("", "<p>html only</p>")
	require.NoError(t, err)
	require.NotEmpty(t, compressed)

	text, html, err := DecompressBody(compressed)
	require.NoError(t, err)
	assert.Empty(t, text)
	assert.Equal(t, "<p>html only</p>", html)
}

func TestDecompressBody_InvalidData(t *testing.T) {
	_, _, err := DecompressBody([]byte("not zstd"))
	assert.Error(t, err)
}
