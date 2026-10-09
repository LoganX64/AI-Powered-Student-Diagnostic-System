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

// NotificationScope selects which rows a viewer may read.
type NotificationScope int

const (
	// ScopeOwn restricts a viewer to rows addressed to them. Used for coaches,
	// who see only their own notifications and never another coach's.
	ScopeOwn NotificationScope = iota
	// ScopeTenant exposes every row in the tenant. Used for the single admin who
	// owns the organization, giving org-wide visibility without a second admin.
	ScopeTenant
)

// scopedClause appends the row-visibility predicate for a scope.
//
// Scope is passed explicitly rather than inferred from whether userID is nil:
// "no user filter" (tenant-wide) and "only my rows" are different permissions,
// and conflating them lets a coach's list widen to the whole org by accident.
//
// A nil userID under ScopeOwn yields a predicate matching nothing, so a caller
// that forgets the identity sees an empty list rather than the whole tenant.
func scopedClause(scope NotificationScope, userID *int, clause string, idx *int, args *[]interface{}) string {
	if scope == ScopeTenant {
		return clause
	}
	*idx++
	own := -1
	if userID != nil {
		own = *userID
	}
	clause += " AND n.user_id = $" + strconv.Itoa(*idx)
	*args = append(*args, own)
	return clause
}

func (r *NotificationRepo) List(tenantID int, userID *int, scope NotificationScope, eventType string, unreadOnly bool, limit, offset int) ([]NotificationRow, int, error) {
	var args []interface{}
	args = append(args, tenantID)
	clause := "WHERE n.tenant_id = $1"
	idx := 1
	clause = scopedClause(scope, userID, clause, &idx, &args)
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

// MarkRead marks one of the caller's own notifications read.
//
// The user_id predicate is load-bearing. Without it any user in a tenant could
// mark anyone's notification read, and because the method reported no
// not-found the handler answered 200 either way. Broadcast rows (user_id IS
// NULL) are deliberately excluded: read_at is a single column, so one user
// marking an org-wide notification read would hide it for every other user.
//
// Returns whether a row was updated, so the handler can distinguish "done" from
// "not yours or not there" instead of reporting success for a no-op.
func (r *NotificationRepo) MarkRead(id, tenantID, userID int) (bool, error) {
	res, err := r.DB.Exec(`UPDATE notifications SET read_at = NOW()
		WHERE id = $1 AND tenant_id = $2 AND user_id = $3`, id, tenantID, userID)
	if err != nil {
		return false, err
	}
	return rowsAffected(res)
}

// MarkAllRead marks the caller's own unread notifications read.
//
// Deliberately owner-scoped, matching MarkRead's rule. It used to include
// broadcast rows (`user_id IS NULL`), which are a single shared row: with the
// admin now holding a tenant-wide view, "mark all as read" would have stamped
// read_at on every coach's rows and emptied their unread badges. It also meant
// the one operation able to clear a broadcast row was reachable by any user,
// contradicting MarkRead's refusal of them.
//
// The admin's view is wider than the set this touches, by design — the rows that
// stay unread are the coaches', and only they may clear them.
func (r *NotificationRepo) MarkAllRead(tenantID int, userID *int) error {
	var args []interface{}
	args = append(args, tenantID)
	idx := 1
	clause := "WHERE tenant_id = $1 AND read_at IS NULL"
	clause = scopedClause(ScopeOwn, userID, clause, &idx, &args)
	_, err := r.DB.Exec(`UPDATE notifications AS n SET read_at = NOW() `+clause, args...)
	return err
}

// Delete removes one of the caller's own notifications. Same ownership rule as
// MarkRead: without the user_id predicate any user in a tenant could delete
// anyone else's, and a tenant-wide broadcast row would be deletable by one
// member. Returns whether a row was removed.
func (r *NotificationRepo) Delete(id, tenantID, userID int) (bool, error) {
	res, err := r.DB.Exec(`DELETE FROM notifications WHERE id = $1 AND tenant_id = $2 AND user_id = $3`, id, tenantID, userID)
	if err != nil {
		return false, err
	}
	return rowsAffected(res)
}

// rowsAffected reports whether an Exec actually touched a row, so callers can
// turn a silent no-op into a not-found. Postgres returns 0 rather than an error
// when a filtered UPDATE/DELETE matches nothing.
func rowsAffected(res sql.Result) (bool, error) {
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// UnreadCount counts unread rows visible under the given scope. The admin's badge
// reflects the whole tenant, a coach's only their own rows.
func (r *NotificationRepo) UnreadCount(tenantID int, userID *int, scope NotificationScope) (int, error) {
	var args []interface{}
	args = append(args, tenantID)
	clause := "WHERE n.tenant_id = $1 AND n.read_at IS NULL"
	idx := 1
	clause = scopedClause(scope, userID, clause, &idx, &args)
	var count int
	if err := r.DB.QueryRow(`SELECT COUNT(*) FROM notifications n `+clause, args...).Scan(&count); err != nil {
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
