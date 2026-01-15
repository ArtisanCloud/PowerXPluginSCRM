package templates

// Template domain HTTP DTOs.

type TemplateListRequest struct {
	Page     int    `form:"page,default=1"`
	PageSize int    `form:"page_size,default=20"`
	Q        string `form:"q"`
}

type CreateTemplateRequest struct {
	Name        string `json:"name"        binding:"required"`
	Description string `json:"description" binding:"required"`
	Content     string `json:"content"     binding:"required"`
}

type UpdateTemplateRequest struct {
	Name        string `json:"name"        binding:"required"`
	Description string `json:"description" binding:"required"`
	Content     string `json:"content"     binding:"required"`
}

type BatchCloneRequest struct {
	SourceIDs         []uint64 `json:"source_ids"        binding:"required"`
	Copies            int      `json:"copies"            binding:"omitempty,min=1,max=50"`
	NamePrefix        string   `json:"name_prefix"`
	DescriptionPrefix string   `json:"description_prefix"`
	Notes             string   `json:"notes"`
}

type ValidateTemplateRequest struct {
	Rules  []string `json:"rules"`
	Strict bool     `json:"strict"`
}
