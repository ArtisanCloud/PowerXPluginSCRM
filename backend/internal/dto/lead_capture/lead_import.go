package lead_capture

// LeadImportError represents a failed row during import.
type LeadImportError struct {
	Row    int    `json:"row"`
	Reason string `json:"reason"`
}

// LeadImportResult returns summary for batch import.
type LeadImportResult struct {
	Total   int               `json:"total"`
	Success int               `json:"success"`
	Failed  int               `json:"failed"`
	Errors  []LeadImportError `json:"errors,omitempty"`
}

// LeadImportPreviewResponse provides header and sample rows for mapping.
type LeadImportPreviewResponse struct {
	Headers           []string       `json:"headers"`
	SampleRows        [][]string     `json:"sample_rows"`
	SuggestedMappings map[string]int `json:"suggested_mappings,omitempty"`
	RequiredFields    []string       `json:"required_fields"`
	AllFields         []string       `json:"all_fields"`
}
