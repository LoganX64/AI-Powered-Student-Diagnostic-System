package repository

import (
	"database/sql"
	"encoding/json"
	"strconv"
)

type NotificationRepo struct {
	DB *sql.DB
}

func NewNotificationRepo(db *sql.DB) *NotificationRepo {
	return &NotificationRepo{DB: db}
}

type NotificationRow struct {
	ID        int             `json:"id"`
	TenantID  int             `json:"tenant_id"`
	UserID    *int            `json:"user_id"`
	EventType string          `json:"event_type"`
	Title     string          `json:"title"`
	Message   string          `json:"message"`
	Priority  string          `json:"priority"`
	ReadAt    *string         `json:"read_at"`
	Metadata  json.RawMessage `json:"metadata"`
	CreatedAt string          `json:"created_at"`
}

type NotificationPrefRow struct {
	ID        int    `json:"id"`
	UserID    int    `json:"user_id"`
	EventType string `json:"event_type"`
	Enabled   bool   `json:"enabled"`
}

func (r *NotificationRepo) List(tenantID int, userID *int, eventType string, unreadOnly bool, limit, offset int) ([]NotificationRow, int, error) {
	var args []interface{}
	args = append(args, tenantID)
	clause := "WHERE n.tenant_id = $1"
	idx := 1
	if userID != nil {
		idx++
		clause += " AND (n.user_id = $" + strconv.Itoa(idx) + " OR n.user_id IS NULL)"
		args = append(args, *userID)
	}
	if eventType != "" {
		idx++
		clause += " AND n.event_type = $" + strconv.Itoa(idx)
		args = append(args, eventType)
	}
	if unreadOnly {
		clause += " AND n.read_at IS NULL"
	}

	countQuery := "SELECT COUNT(*) FROM notifications n " + clause
	var total int
	if err := r.DB.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	idx++
	query := "SELECT n.id, n.tenant_id, n.user_id, n.event_type, n.title, n.message, n.priority, n.read_at, n.metadata, n.created_at FROM notifications n " +
		clause + " ORDER BY n.created_at DESC"
	query += " LIMIT $" + strconv.Itoa(idx)
	args = append(args, limit)
	idx++
	query += " OFFSET $" + strconv.Itoa(idx)
	args = append(args, offset)

	rows, err := r.DB.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := []NotificationRow{}
	for rows.Next() {
		var n NotificationRow
		var userID sql.NullInt64
		var readAt sql.NullString
		var meta []byte
		if err := rows.Scan(&n.ID, &n.TenantID, &userID, &n.EventType, &n.Title, &n.Message, &n.Priority, &readAt, &meta, &n.CreatedAt); err != nil {
			return nil, 0, err
		}
		if userID.Valid {
			u := int(userID.Int64)
			n.UserID = &u
		}
		if readAt.Valid {
			ra := readAt.String
			n.ReadAt = &ra
		}
		n.Metadata = json.RawMessage(meta)
		out = append(out, n)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	if out == nil {
		out = []NotificationRow{}
	}
	return out, total, nil
}

func (r *NotificationRepo) GetByID(id, tenantID int) (*NotificationRow, error) {
	var n NotificationRow
	var userID sql.NullInt64
	var readAt sql.NullString
	var meta []byte
	query := `SELECT id, tenant_id, user_id, event_type, title, message, priority, read_at, metadata, created_at FROM notifications WHERE id = $1 AND tenant_id = $2`
	if err := r.DB.QueryRow(query, id, tenantID).Scan(&n.ID, &n.TenantID, &userID, &n.EventType, &n.Title, &n.Message, &n.Priority, &readAt, &meta, &n.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if userID.Valid {
		u := int(userID.Int64)
		n.UserID = &u
	}
	if readAt.Valid {
		ra := readAt.String
		n.ReadAt = &ra
	}
	n.Metadata = json.RawMessage(meta)
	return &n, nil
}

func (r *NotificationRepo) Create(n NotificationRow) (int, error) {
	var id int
	query := `INSERT INTO notifications (tenant_id, user_id, event_type, title, message, priority, metadata) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`
	if err := r.DB.QueryRow(query, n.TenantID, n.UserID, n.EventType, n.Title, n.Message, n.Priority, n.Metadata).Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

func (r *NotificationRepo) MarkRead(id, tenantID int) error {
	_, err := r.DB.Exec(`UPDATE notifications SET read_at = NOW() WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	return err
}

func (r *NotificationRepo) MarkAllRead(tenantID int, userID *int) error {
	if userID != nil {
		_, err := r.DB.Exec(`UPDATE notifications SET read_at = NOW() WHERE tenant_id = $1 AND read_at IS NULL AND (user_id = $2 OR user_id IS NULL)`, tenantID, *userID)
		return err
	}
	_, err := r.DB.Exec(`UPDATE notifications SET read_at = NOW() WHERE tenant_id = $1 AND read_at IS NULL`, tenantID)
	return err
}

func (r *NotificationRepo) Delete(id, tenantID int) error {
	_, err := r.DB.Exec(`DELETE FROM notifications WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	return err
}

func (r *NotificationRepo) UnreadCount(tenantID int, userID *int) (int, error) {
	var count int
	var err error
	if userID != nil {
		err = r.DB.QueryRow(`SELECT COUNT(*) FROM notifications WHERE tenant_id = $1 AND read_at IS NULL AND (user_id = $2 OR user_id IS NULL)`, tenantID, *userID).Scan(&count)
	} else {
		err = r.DB.QueryRow(`SELECT COUNT(*) FROM notifications WHERE tenant_id = $1 AND read_at IS NULL`, tenantID).Scan(&count)
	}
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *NotificationRepo) GetPreferences(userID int) ([]NotificationPrefRow, error) {
	rows, err := r.DB.Query(`SELECT id, user_id, event_type, enabled FROM notification_preferences WHERE user_id = $1 ORDER BY event_type`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []NotificationPrefRow{}
	for rows.Next() {
		var p NotificationPrefRow
		if err := rows.Scan(&p.ID, &p.UserID, &p.EventType, &p.Enabled); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if out == nil {
		out = []NotificationPrefRow{}
	}
	return out, nil
}

func (r *NotificationRepo) UpdatePreferences(userID int, prefs map[string]bool) error {
	for eventType, enabled := range prefs {
		if _, err := r.DB.Exec(
			`INSERT INTO notification_preferences (user_id, event_type, enabled) VALUES ($1, $2, $3)
			 ON CONFLICT (user_id, event_type) DO UPDATE SET enabled = $3, updated_at = NOW()`,
			userID, eventType, enabled,
		); err != nil {
			return err
		}
	}
	return nil
}

func (r *NotificationRepo) IsEventEnabled(userID int, eventType string) (bool, error) {
	var enabled bool
	err := r.DB.QueryRow(`SELECT enabled FROM notification_preferences WHERE user_id = $1 AND event_type = $2`, userID, eventType).Scan(&enabled)
	if err == sql.ErrNoRows {
		// No preference row means the event is enabled by default.
		return true, nil
	}
	if err != nil {
		return true, err
	}
	return enabled, nil
}
