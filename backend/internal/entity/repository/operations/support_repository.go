package operations

import (
	"context"
	"errors"
	"time"

	opmodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/operations"
	repository "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SupportRepository encapsulates persistence for support channels, tickets, and readiness items.
type SupportRepository struct {
	db            *gorm.DB
	channelsRepo  *repository.BaseRepository[opmodels.SupportChannel]
	ticketsRepo   *repository.BaseRepository[opmodels.SupportTicket]
	eventsRepo    *repository.BaseRepository[opmodels.SupportTicketEvent]
	readinessRepo *repository.BaseRepository[opmodels.ReadinessChecklistItem]
}

// NewSupportRepository constructs a repository with shared DB handle.
func NewSupportRepository(db *gorm.DB) *SupportRepository {
	return &SupportRepository{
		db:            db,
		channelsRepo:  repository.NewBaseRepository[opmodels.SupportChannel](db),
		ticketsRepo:   repository.NewBaseRepository[opmodels.SupportTicket](db),
		eventsRepo:    repository.NewBaseRepository[opmodels.SupportTicketEvent](db),
		readinessRepo: repository.NewBaseRepository[opmodels.ReadinessChecklistItem](db),
	}
}

// UpsertChannel saves a support channel configuration.
func (r *SupportRepository) UpsertChannel(ctx context.Context, channel *opmodels.SupportChannel) (*opmodels.SupportChannel, error) {
	if channel.ID == "" {
		channel.ID = uuid.NewString()
	}
	channel.UpdatedAt = time.Now().UTC()
	if channel.CreatedAt.IsZero() {
		channel.CreatedAt = time.Now().UTC()
	}
	if err := r.db.WithContext(ctx).Save(channel).Error; err != nil {
		return nil, err
	}
	return channel, nil
}

// ListChannels returns support channels for the given scope.
func (r *SupportRepository) ListChannels(ctx context.Context, pluginID string, tenantID *string) ([]*opmodels.SupportChannel, error) {
	query := r.db.WithContext(ctx).Where("plugin_id = ?", pluginID)
	if tenantID != nil {
		query = query.Where("tenant_uuid = ?", tenantID)
	} else {
		query = query.Where("tenant_uuid IS NULL")
	}
	var channels []*opmodels.SupportChannel
	if err := query.Order("channel").Find(&channels).Error; err != nil {
		return nil, err
	}
	return channels, nil
}

// DeleteChannels removes channels matching scope (used during bulk updates).
func (r *SupportRepository) DeleteChannels(ctx context.Context, pluginID string, tenantID *string) error {
	query := r.db.WithContext(ctx).Where("plugin_id = ?", pluginID)
	if tenantID != nil {
		query = query.Where("tenant_uuid = ?", tenantID)
	} else {
		query = query.Where("tenant_uuid IS NULL")
	}
	return query.Delete(&opmodels.SupportChannel{}).Error
}

// CreateTicket creates a support ticket record.
func (r *SupportRepository) CreateTicket(ctx context.Context, ticket *opmodels.SupportTicket) (*opmodels.SupportTicket, error) {
	if ticket.ID == "" {
		ticket.ID = uuid.NewString()
	}
	if ticket.CreatedAt.IsZero() {
		ticket.CreatedAt = time.Now().UTC()
	}
	ticket.UpdatedAt = ticket.CreatedAt
	if _, err := r.ticketsRepo.Create(ctx, ticket); err != nil {
		return nil, err
	}
	return ticket, nil
}

// UpdateTicket updates ticket fields.
func (r *SupportRepository) UpdateTicket(ctx context.Context, ticket *opmodels.SupportTicket) error {
	ticket.UpdatedAt = time.Now().UTC()
	_, err := r.ticketsRepo.Update(ctx, ticket)
	return err
}

// FindTicket fetches ticket by ID and plugin scope.
func (r *SupportRepository) FindTicket(ctx context.Context, pluginID, ticketID string) (*opmodels.SupportTicket, error) {
	var ticket opmodels.SupportTicket
	err := r.db.WithContext(ctx).
		Where("id = ? AND plugin_id = ?", ticketID, pluginID).
		First(&ticket).Error
	if err != nil {
		return nil, err
	}
	return &ticket, nil
}

// AppendEvent records ticket event.
func (r *SupportRepository) AppendEvent(ctx context.Context, event *opmodels.SupportTicketEvent) (*opmodels.SupportTicketEvent, error) {
	if event.EmittedAt.IsZero() {
		event.EmittedAt = time.Now().UTC()
	}
	event.CreatedAt = time.Now().UTC()
	if _, err := r.eventsRepo.Create(ctx, event); err != nil {
		return nil, err
	}
	return event, nil
}

// ListEvents returns recent events for ticket.
func (r *SupportRepository) ListEvents(ctx context.Context, ticketID string) ([]*opmodels.SupportTicketEvent, error) {
	var events []*opmodels.SupportTicketEvent
	if err := r.db.WithContext(ctx).
		Where("ticket_id = ?", ticketID).
		Order("emitted_at DESC").
		Find(&events).Error; err != nil {
		return nil, err
	}
	return events, nil
}

// ListTickets returns tickets for the plugin scope.
func (r *SupportRepository) ListTickets(ctx context.Context, pluginID string) ([]*opmodels.SupportTicket, error) {
	var tickets []*opmodels.SupportTicket
	if err := r.db.WithContext(ctx).
		Where("plugin_id = ?", pluginID).
		Find(&tickets).Error; err != nil {
		return nil, err
	}
	return tickets, nil
}

// UpsertReadinessItem inserts or updates readiness checklist item.
func (r *SupportRepository) UpsertReadinessItem(ctx context.Context, item *opmodels.ReadinessChecklistItem) (*opmodels.ReadinessChecklistItem, error) {
	if item.ID == "" {
		item.ID = uuid.NewString()
	}
	item.UpdatedAt = time.Now().UTC()
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now().UTC()
	}
	if err := r.db.WithContext(ctx).Save(item).Error; err != nil {
		return nil, err
	}
	return item, nil
}

// ListReadinessByType fetches readiness items.
func (r *SupportRepository) ListReadinessByType(ctx context.Context, pluginID, checklistType string) ([]*opmodels.ReadinessChecklistItem, error) {
	var items []*opmodels.ReadinessChecklistItem
	if err := r.db.WithContext(ctx).
		Where("plugin_id = ? AND type = ?", pluginID, checklistType).
		Order("item_key").
		Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// ErrTicketNotFound indicates missing ticket.
var ErrTicketNotFound = errors.New("support ticket not found")
