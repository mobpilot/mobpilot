package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mobpilot/mobpilot/services/notification/domain"
	domainports "github.com/mobpilot/mobpilot/services/notification/domain/ports"
	sqlcn "github.com/mobpilot/mobpilot/services/notification/infrastructure/postgres/sqlc"
)

// NotificationRepo is the PostgreSQL adapter implementing domain/ports.NotificationRepository.
type NotificationRepo struct {
	pool    *pgxpool.Pool
	queries *sqlcn.Queries
}

var _ domainports.NotificationRepository = (*NotificationRepo)(nil)

func NewNotificationRepo(pool *pgxpool.Pool) *NotificationRepo {
	return &NotificationRepo{pool: pool, queries: sqlcn.New(pool)}
}

func (r *NotificationRepo) Save(ctx context.Context, n *domain.Notification) error {
	data, err := json.Marshal(n.Data)
	if err != nil {
		return fmt.Errorf("NotificationRepo.Save marshal data: %w", err)
	}

	var groupID, orgID pgtype.UUID
	if n.GroupID != nil {
		groupID = pgtype.UUID{Bytes: [16]byte(*n.GroupID), Valid: true}
	}
	if n.OrgID != nil {
		orgID = pgtype.UUID{Bytes: [16]byte(*n.OrgID), Valid: true}
	}

	if err := r.queries.InsertNotification(ctx, sqlcn.InsertNotificationParams{
		ID:             n.ID,
		UserID:         n.UserID,
		AppID:          n.AppID,
		GroupID:        groupID,
		OrgID:          orgID,
		Title:          n.Title,
		Body:           n.Body,
		Data:           data,
		DeliveryStatus: string(n.DeliveryStatus),
		CreatedAt:      n.CreatedAt,
	}); err != nil {
		return fmt.Errorf("NotificationRepo.Save: %w", err)
	}
	return nil
}

func (r *NotificationRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Notification, error) {
	row, err := r.queries.GetNotificationByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotificationNotFound
		}
		return nil, fmt.Errorf("NotificationRepo.FindByID: %w", err)
	}
	return rowToNotification(row), nil
}

func (r *NotificationRepo) ListForUser(ctx context.Context, userID, appID uuid.UUID, limit, offset int) ([]*domain.Notification, error) {
	rows, err := r.queries.ListNotificationsForUser(ctx, sqlcn.ListNotificationsForUserParams{
		UserID: userID,
		AppID:  appID,
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("NotificationRepo.ListForUser: %w", err)
	}
	ns := make([]*domain.Notification, len(rows))
	for i, row := range rows {
		ns[i] = rowToNotification(row)
	}
	return ns, nil
}

func (r *NotificationRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.DeliveryStatus) error {
	if err := r.queries.UpdateNotificationStatus(ctx, sqlcn.UpdateNotificationStatusParams{
		ID:             id,
		DeliveryStatus: string(status),
	}); err != nil {
		return fmt.Errorf("NotificationRepo.UpdateStatus: %w", err)
	}
	return nil
}

func (r *NotificationRepo) MarkRead(ctx context.Context, id uuid.UUID) error {
	if err := r.queries.MarkNotificationRead(ctx, id); err != nil {
		return fmt.Errorf("NotificationRepo.MarkRead: %w", err)
	}
	return nil
}

func rowToNotification(row sqlcn.Notification) *domain.Notification {
	n := &domain.Notification{
		ID:             row.ID,
		UserID:         row.UserID,
		AppID:          row.AppID,
		Title:          row.Title,
		Body:           row.Body,
		DeliveryStatus: domain.DeliveryStatus(row.DeliveryStatus),
		CreatedAt:      row.CreatedAt,
	}
	if len(row.Data) > 0 {
		_ = json.Unmarshal(row.Data, &n.Data)
	}
	if n.Data == nil {
		n.Data = map[string]any{}
	}
	if row.GroupID.Valid {
		id := uuid.UUID(row.GroupID.Bytes)
		n.GroupID = &id
	}
	if row.OrgID.Valid {
		id := uuid.UUID(row.OrgID.Bytes)
		n.OrgID = &id
	}
	if row.ReadAt.Valid {
		t := row.ReadAt.Time
		n.ReadAt = &t
	}
	return n
}
