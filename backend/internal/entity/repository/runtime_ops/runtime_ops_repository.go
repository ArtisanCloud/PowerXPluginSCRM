package runtime_ops

import "context"

// AssignmentRepository defines storage behaviours for runtime assignment lifecycle.
type AssignmentRepository interface {
	CreateAssignment(ctx context.Context, assignment *RuntimeAssignment) error
}

// RuntimeAssignment is a placeholder domain model used by the repository scaffolding.
type RuntimeAssignment struct{}
