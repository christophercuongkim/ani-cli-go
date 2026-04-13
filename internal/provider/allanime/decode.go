package allanime

import "encoding/hex"

// DecodeSourceURL decodes AllAnime's obfuscated source URLs
// URLs are hex-encoded and XOR'd with 0x38
func DecodeSourceURL(encoded string) (string, error) {
	if encoded == "" {
		return "", nil
	}

	decoded, err := hex.DecodeString(encoded)
	if err != nil {
		return "", ErrInvalidHexString
	}

	result := make([]byte, len(decoded))
	for i, b := range decoded {
		result[i] = b ^ 0x38
	}

	return string(result), nil
}
