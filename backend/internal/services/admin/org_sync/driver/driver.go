package driver

import "context"

// AccountContext describes a channel account for org sync.
type AccountContext struct {
	TenantUUID         string
	ChannelAccountUUID string
	ChannelCode        string
	AppType            string
	AccountID          string
	DisplayName        string
	Credentials        map[string]string
}

// SourceUnitDTO normalizes a source department.
type SourceUnitDTO struct {
	ExternalUnitID       string
	ParentExternalUnitID *string
	Name                 string
	Order                int
	Status               string
}

// SourceMemberDTO normalizes a source member.
type SourceMemberDTO struct {
	ExternalMemberID string
	Name             string
	Phone            string
	Email            string
	BizMail          string
	Position         string
	Address          string
	MainDepartmentID string
	DepartmentIDs    []string
	DepartmentOrders map[string]int
	AvatarURL        string
	ProfileStatus    string
	Status           string
}

// OrgSyncDriver abstracts channel-specific sync.
type OrgSyncDriver interface {
	FetchUnits(ctx context.Context, account AccountContext) ([]SourceUnitDTO, error)
	FetchMembers(ctx context.Context, account AccountContext) ([]SourceMemberDTO, error)
}
