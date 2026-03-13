package lead_capture

// LeadSourceCatalogCreateRequest defines payload for creating a source catalog item.
type LeadSourceCatalogCreateRequest struct {
	Category string `json:"category" binding:"required,oneof=traffic_platform traffic_source"`
	Code     string `json:"code" binding:"required"`
	Label    string `json:"label" binding:"required"`
	Sort     int    `json:"sort" binding:"omitempty"`
	Enabled  *bool  `json:"enabled" binding:"omitempty"`
}

// LeadSourceCatalogUpdateRequest defines payload for updating a source catalog item.
type LeadSourceCatalogUpdateRequest struct {
	Code    string `json:"code" binding:"omitempty"`
	Label   string `json:"label" binding:"omitempty"`
	Sort    *int   `json:"sort" binding:"omitempty"`
	Enabled *bool  `json:"enabled" binding:"omitempty"`
}
