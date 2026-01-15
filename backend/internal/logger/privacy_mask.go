package logger

import "strings"

var (
	maskFieldSet    = map[string]struct{}{}
	maskPlaceholder = "[REDACTED]"
)

// ConfigurePrivacyMasker configures runtime field masking based on provided keys.
// Field names are normalised to lower-case for comparisons.
func ConfigurePrivacyMasker(fields []string, placeholder string) {
	maskFieldSet = make(map[string]struct{}, len(fields))
	for _, f := range fields {
		if f = strings.TrimSpace(f); f != "" {
			maskFieldSet[strings.ToLower(f)] = struct{}{}
		}
	}
	if placeholder != "" {
		maskPlaceholder = placeholder
	}
	RegisterRuntimeMasker(applyPrivacyMask)
}

func applyPrivacyMask(fields Fields) Fields {
	if len(maskFieldSet) == 0 || len(fields) == 0 {
		return fields
	}
	for key := range fields {
		if _, ok := maskFieldSet[strings.ToLower(key)]; ok {
			fields[key] = maskPlaceholder
		}
	}
	return fields
}
