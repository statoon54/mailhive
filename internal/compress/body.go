package compress

import (
	"encoding/json"
	"fmt"

	"github.com/klauspost/compress/zstd"
)

// compressedBody est la structure interne sérialisée avant compression.
type compressedBody struct {
	TextBody string `json:"t,omitempty"`
	HTMLBody string `json:"h,omitempty"`
}

var (
	encoder, _ = zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedDefault))
	decoder, _ = zstd.NewReader(nil)
)

// CompressBody compresse le text_body et html_body en JSON+zstd.
func CompressBody(textBody, htmlBody string) ([]byte, error) {
	if textBody == "" && htmlBody == "" {
		return nil, nil
	}
	data, err := json.Marshal(compressedBody{TextBody: textBody, HTMLBody: htmlBody})
	if err != nil {
		return nil, fmt.Errorf("erreur de sérialisation du corps : %w", err)
	}
	return encoder.EncodeAll(data, make([]byte, 0, len(data)/2)), nil
}

// DecompressBody décompresse un corps compressé et retourne (textBody, htmlBody).
func DecompressBody(compressed []byte) (string, string, error) {
	if len(compressed) == 0 {
		return "", "", nil
	}
	data, err := decoder.DecodeAll(compressed, nil)
	if err != nil {
		return "", "", fmt.Errorf("erreur de décompression du corps : %w", err)
	}
	var body compressedBody
	if err := json.Unmarshal(data, &body); err != nil {
		return "", "", fmt.Errorf("erreur de désérialisation du corps : %w", err)
	}
	return body.TextBody, body.HTMLBody, nil
}
