package capability

// Type enumerates the transport or surface that exposes the capability.
type Type string

const (
	TypeAPI    Type = "API"
	TypeEvent  Type = "Event"
	TypeJob    Type = "Job"
	TypeTool   Type = "Tool"
	TypeBridge Type = "Bridge"
	TypeSchema Type = "Schema"
)

// Status tracks the lifecycle of a capability descriptor.
type Status string

const (
	StatusActive     Status = "active"
	StatusDeprecated Status = "deprecated"
	StatusSunset     Status = "sunset"
)

// Descriptor is the machine readable definition of a plugin capability.
type Descriptor struct {
	ID          string                 `yaml:"id" json:"id"`
	Type        Type                   `yaml:"type" json:"type"`
	Version     string                 `yaml:"version" json:"version"`
	Status      Status                 `yaml:"status,omitempty" json:"status,omitempty"`
	Description string                 `yaml:"description,omitempty" json:"description,omitempty"`
	RBAC        RBAC                   `yaml:"rbac" json:"rbac"`
	Provides    []SchemaRef            `yaml:"provides,omitempty" json:"provides,omitempty"`
	Consumes    []SchemaRef            `yaml:"consumes,omitempty" json:"consumes,omitempty"`
	Metadata    map[string]interface{} `yaml:"metadata,omitempty" json:"metadata,omitempty"`
}

// RBAC binds the capability to RBAC resources/actions.
type RBAC struct {
	Resource string   `yaml:"resource" json:"resource"`
	Actions  []string `yaml:"actions" json:"actions"`
}

// SchemaRef points to a JSON Schema that documents payload shape.
type SchemaRef struct {
	ID      string `yaml:"id" json:"id"`
	Path    string `yaml:"path,omitempty" json:"path,omitempty"`
	Kind    string `yaml:"kind,omitempty" json:"kind,omitempty"`
	Version string `yaml:"version,omitempty" json:"version,omitempty"`
}
