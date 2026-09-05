package sercomm

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

// ComputePassword hashes the plaintext password according to the Sercomm
// two-step HMAC-SHA256 scheme:
//  1. h1 = HMAC-SHA256(key: "$1$SERCOMM$", data: password)
//  2. h2 = HMAC-SHA256(key: encryptionKey, data: hex(h1))
//
// The result is returned as a lowercase hex string.
func ComputePassword(password, encryptionKey string) string {
	h1 := hmac.New(sha256.New, []byte("$1$SERCOMM$"))
	h1.Write([]byte(password))
	hash1 := hex.EncodeToString(h1.Sum(nil))

	h2 := hmac.New(sha256.New, []byte(encryptionKey))
	h2.Write([]byte(hash1))
	return hex.EncodeToString(h2.Sum(nil))
}

// ParseUserLang parses the array of single-key objects returned by
// /data/user_lang.json into a flat key-value map.
//
// Example input:
//
//	[{"fw_version": "AT904X-03.01s"}, {"wan_ip4_addr": "217.34.98.17s"}, ...]
func ParseUserLang(data []byte) (map[string]string, error) {
	var list []map[string]any
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, fmt.Errorf("unmarshal user_lang: %w", err)
	}
	out := make(map[string]string, len(list))
	for _, item := range list {
		for k, v := range item {
			s, ok := v.(string)
			if !ok {
				s = fmt.Sprintf("%v", v)
			}
			out[k] = cleanSercommString(s)
		}
	}
	return out, nil
}

// cleanSercommString removes the trailing 's' type marker commonly
// emitted by Sercomm JSON endpoints (e.g. "AT904X-03.01s" -> "AT904X-03.01").
func cleanSercommString(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 1 && strings.HasSuffix(s, "s") && !strings.Contains(s, " ") {
		return strings.TrimSuffix(s, "s")
	}
	return s
}
