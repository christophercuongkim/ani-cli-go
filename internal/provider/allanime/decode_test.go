package allanime

import (
	"fmt"
	"testing"
)

func TestDecodeSourceURL(t *testing.T) {
	tests := []struct {
		name    string
		encoded string
		want    string
		wantErr bool
	}{
		{
			name:    "empty string",
			encoded: "",
			want:    "",
			wantErr: false,
		},
		{
			name:    "simple decode",
			encoded: "59",
			want:    "a",
			wantErr: false,
		},
		{
			name:    "decode hello",
			encoded: "505d545457",
			want:    "hello",
			wantErr: false,
		},
		{
			name:    "decode URL with slashes",
			encoded: "504c4c484b0217175d40595548545d165b5755",
			want:    "https://example.com",
			wantErr: false,
		},
		{
			name:    "invalid hex - odd length",
			encoded: "5",
			wantErr: true,
		},
		{
			name:    "invalid hex - bad chars",
			encoded: "zz",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DecodeSourceURL(tt.encoded)
			if (err != nil) != tt.wantErr {
				t.Errorf("DecodeSourceURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("DecodeSourceURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDecodeSourceURL_RoundTrip(t *testing.T) {
	// Verify XOR 0x38 is reversible
	inputs := []string{
		"hello",
		"https://example.com/path?query=1",
		"wixmp.repackager.com",
		"12345",
	}

	for _, input := range inputs {
		// Encode manually
		encoded := encodeForTest(input)

		// Decode
		decoded, err := DecodeSourceURL(encoded)
		if err != nil {
			t.Errorf("DecodeSourceURL(%q) unexpected error: %v", encoded, err)
			continue
		}

		if decoded != input {
			t.Errorf("Round trip failed: input=%q, encoded=%q, decoded=%q", input, encoded, decoded)
		}
	}
}

func encodeForTest(s string) string {
	result := ""
	for i := 0; i < len(s); i++ {
		result += fmt.Sprintf("%02x", s[i]^0x38)
	}
	return result
}
