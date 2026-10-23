package repository

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
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
	TenantID  *int            `json:"tenant_id"`
	UserID    *int            `json:"user_id"`
	EventType string          `json:"event_type"`
	Title     string          `json:"title"`
	Message   string          `json:"message"`
	Priority  string          `json:"priority"`
	Scope     string          `json:"scope"`
	ReadAt    *string         `json:"read_at"`
	SeenAt    *string         `json:"seen_at"`
	EventKey  *string         `json:"event_key"`
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
	// ScopeOwn restricts a viewer to rows addressed to them. Used for tenant
	// admins and coaches, who see only their own notifications.
	ScopeOwn NotificationScope = iota
	// ScopeTenant exposes every row in the tenant. Retained only for the
	// pre-phase-4 admin view; new code should use ScopeOwn.
	ScopeTenant
	// ScopePlatform restricts a super admin to platform rows, which have no
	// tenant_id.
	ScopePlatform
)

// NotificationFilter describes one page of notifications. Read and Seen are
// tri-state: nil means "any", true means "must be set", false means "must be
// NULL".
type NotificationFilter struct {
	TenantID  *int
	UserID    int
	Scope     NotificationScope
	EventType string
	Priority  string
	Read      *bool
	Seen      *bool
	Limit     int
	Offset    int
}

// NotificationCounts is the pair the bell needs: unseen drives the badge, unread
// drives the Unread tab.
type NotificationCounts struct {
	Unread int
	Unseen int
}

// scopePredicate returns the row-visibility predicate for a scope. The
// placeholders start after idOffset so the same predicate can follow an id
// argument in single-row queries.
func scopePredicate(scope NotificationScope, tenantID *int, userID, idOffset int) (string, []interface{}, error) {
	switch scope {
	case ScopePlatform:
		return "n.scope = 'platform' AND n.tenant_id IS NULL AND n.user_id = $" + strconv.Itoa(idOffset+1),
			[]interface{}{userID}, nil
	case ScopeTenant:
		if tenantID == nil {
			return "", nil, errors.New("tenant id is required for tenant-scoped notifications")
		}
		return "n.scope = 'tenant' AND n.tenant_id = $" + strconv.Itoa(idOffset+1),
			[]interface{}{*tenantID}, nil
	case ScopeOwn:
		if tenantID == nil {
			return "", nil, errors.New("tenant id is required for tenant-scoped notifications")
		}
		return "n.scope = 'tenant' AND n.tenant_id = $" + strconv.Itoa(idOffset+1) +
				" AND n.user_id = $" + strconv.Itoa(idOffset+2),
			[]interface{}{*tenantID, userID}, nil
	default:
		return "", nil, fmt.Errorf("unknown notification scope %d", scope)
	}
}

// actionPredicate is the write-side counterpart to scopePredicate. Seens, reads,
// and deletes are always owner-scoped: even the legacy ScopeTenant listing is
// wider than the set a caller may mutate, and one user must never clear another
// user's badge or delete their row.
func actionPredicate(scope NotificationScope, tenantID *int, userID, idOffset int) (string, []interface{}, error) {
	switch scope {
	case ScopePlatform:
		return "n.scope = 'platform' AND n.tenant_id IS NULL AND n.user_id = $" + strconv.Itoa(idOffset+1),
			[]interface{}{userID}, nil
	case ScopeOwn, ScopeTenant:
		if tenantID == nil {
			return "", nil, errors.New("tenant id is required for tenant-scoped notifications")
		}
		return "n.scope = 'tenant' AND n.tenant_id = $" + strconv.Itoa(idOffset+1) +
				" AND n.user_id = $" + strconv.Itoa(idOffset+2),
			[]interface{}{*tenantID, userID}, nil
	default:
		return "", nil, fmt.Errorf("unknown notification scope %d", scope)
	}
}

// ListFiltered returns a page of notifications plus the total matching the
// filters.
func (r *NotificationRepo) ListFiltered(f NotificationFilter) ([]NotificationRow, int, error) {
	where, args, err := scopePredicate(f.Scope, f.TenantID, f.UserID, 0)
	if err != nil {
		return nil, 0, err
	}
	idx := len(args)

	if f.EventType != "" {
		idx++
		where += " AND n.event_type = $" + strconv.Itoa(idx)
		args = append(args, f.EventType)
	}
	if f.Priority != "" {
		idx++
		where += " AND n.priority = $" + strconv.Itoa(idx)
		args = append(args, f.Priority)
	}
	if f.Read != nil {
		if *f.Read {
			where += " AND n.read_at IS NOT NULL"
		} else {
			where += " AND n.read_at IS NULL"
		}
	}
	if f.Seen != nil {
		if *f.Seen {
			where += " AND n.seen_at IS NOT NULL"
		} else {
			where += " AND n.seen_at IS NULL"
		}
	}

	var total int
	if err := r.DB.QueryRow(`SELECT COUNT(*) FROM notifications n WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limit := f.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := f.Offset
	if offset < 0 {
		offset = 0
	}

	idx++
	query := `SELECT n.id, n.tenant_id, n.user_id, n.event_type, n.title, n.message,
		n.priority, n.scope, n.read_at, n.seen_at, n.event_key, n.metadata, n.created_at
		FROM notifications n WHERE ` + where + ` ORDER BY n.created_at DESC LIMIT $` + strconv.Itoa(idx)
	args = append(args, limit)
	idx++
	query += ` OFFSET $` + strconv.Itoa(idx)
	args = append(args, offset)

	rows, err := r.DB.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := []NotificationRow{}
	for rows.Next() {
		n, err := scanNotificationRow(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, n)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

// List is the legacy positional wrapper. Keep it until the handlers move to
// NotificationFilter in phase 4.
func (r *NotificationRepo) List(tenantID int, userID *int, scope NotificationScope, eventType, priority string, unreadOnly bool, limit, offset int) ([]NotificationRow, int, error) {
	uid := -1
	if userID != nil {
		uid = *userID
	}
	f := NotificationFilter{
		TenantID:  &tenantID,
		UserID:    uid,
		Scope:     scope,
		EventType: eventType,
		Priority:  priority,
		Limit:     limit,
		Offset:    offset,
	}
	if unreadOnly {
		unread := false
		f.Read = &unread
	}
	return r.ListFiltered(f)
}

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanNotificationRow(scanner rowScanner) (NotificationRow, error) {
	var n NotificationRow
	var tenantID, userID sql.NullInt64
	var readAt, seenAt, eventKey sql.NullString
	var meta []byte

	if err := scanner.Scan(
		&n.ID, &tenantID, &userID, &n.EventType, &n.Title, &n.Message,
		&n.Priority, &n.Scope, &readAt, &seenAt, &eventKey, &meta, &n.CreatedAt,
	); err != nil {
		return n, err
	}

	if tenantID.Valid {
		v := int(tenantID.Int64)
		n.TenantID = &v
	}
	if userID.Valid {
		v := int(userID.Int64)
		n.UserID = &v
	}
	if readAt.Valid {
		v := readAt.String
		n.ReadAt = &v
	}
	if seenAt.Valid {
		v := seenAt.String
		n.SeenAt = &v
	}
	if eventKey.Valid {
		v := eventKey.String
		n.EventKey = &v
	}
	n.Metadata = json.RawMessage(meta)
	return n, nil
}

func (r *NotificationRepo) GetByID(id, tenantID int) (*NotificationRow, error) {
	where, args, err := scopePredicate(ScopeTenant, &tenantID, 0, 1)
	if err != nil {
		return nil, err
	}
	query := `SELECT n.id, n.tenant_id, n.user_id, n.event_type, n.title, n.message,
		n.priority, n.scope, n.read_at, n.seen_at, n.event_key, n.metadata, n.created_at
		FROM notifications n WHERE n.id = $1 AND ` + where
	args = append([]interface{}{id}, args...)

	n, err := scanNotificationRow(r.DB.QueryRow(query, args...))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &n, nil
}

func (r *NotificationRepo) Create(n NotificationRow) (int, error) {
	id, _, err := r.CreateIdempotent(n)
	return id, err
}

// CreateIdempotent inserts a notification unless the same user already has a row
// with the same event_key. A conflict is reported as created=false, not as an
// error, so producers can retry safely.
func (r *NotificationRepo) CreateIdempotent(n NotificationRow) (int, bool, error) {
	if n.UserID == nil {
		return 0, false, errors.New("notification user_id is required")
	}
	if n.Scope == "" {
		n.Scope = "tenant"
	}
	switch n.Scope {
	case "tenant":
		if n.TenantID == nil {
			return 0, false, errors.New("tenant notification requires tenant_id")
		}
	case "platform":
		if n.TenantID != nil {
			return 0, false, errors.New("platform notification must not set tenant_id")
		}
	default:
		return 0, false, fmt.Errorf("invalid notification scope %q", n.Scope)
	}

	meta := n.Metadata
	if len(meta) == 0 {
		meta = json.RawMessage("{}")
	}
	eventKey := n.EventKey
	if eventKey != nil && *eventKey == "" {
		eventKey = nil
	}

	var id int
	err := r.DB.QueryRow(`
		INSERT INTO notifications (
			tenant_id, user_id, event_type, title, message,
			priority, metadata, scope, event_key
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (user_id, event_key) WHERE event_key IS NOT NULL DO NOTHING
		RETURNING id`,
		n.TenantID, *n.UserID, n.EventType, n.Title, n.Message,
		n.Priority, meta, n.Scope, eventKey,
	).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return id, true, nil
}

// MarkReadScoped marks one of the caller's own notifications read. The scope
// controls whether the row must be a tenant row or a platform row; ownership is
// always enforced. Marking read also marks seen.
func (r *NotificationRepo) MarkReadScoped(id int, tenantID *int, userID int, scope NotificationScope) (bool, error) {
	where, args, err := actionPredicate(scope, tenantID, userID, 1)
	if err != nil {
		return false, err
	}
	query := `UPDATE notifications AS n
		SET read_at = COALESCE(read_at, NOW()), seen_at = COALESCE(seen_at, NOW())
		WHERE n.id = $1 AND ` + where
	args = append([]interface{}{id}, args...)
	res, err := r.DB.Exec(query, args...)
	if err != nil {
		return false, err
	}
	return rowsAffected(res)
}

func (r *NotificationRepo) MarkRead(id, tenantID, userID int) (bool, error) {
	return r.MarkReadScoped(id, &tenantID, userID, ScopeOwn)
}

// MarkAllReadScoped marks the caller's own unread notifications read. It is
// owner-scoped even when the legacy ScopeTenant listing is wider.
func (r *NotificationRepo) MarkAllReadScoped(tenantID *int, userID int, scope NotificationScope) error {
	where, args, err := actionPredicate(scope, tenantID, userID, 0)
	if err != nil {
		return err
	}
	_, err = r.DB.Exec(`UPDATE notifications AS n
		SET read_at = COALESCE(read_at, NOW()), seen_at = COALESCE(seen_at, NOW())
		WHERE n.read_at IS NULL AND `+where, args...)
	return err
}

func (r *NotificationRepo) MarkAllRead(tenantID int, userID *int) error {
	if userID == nil {
		return errors.New("notification user_id is required")
	}
	return r.MarkAllReadScoped(&tenantID, *userID, ScopeOwn)
}

// MarkSeenScoped marks one of the caller's own notifications seen without
// marking it read.
func (r *NotificationRepo) MarkSeenScoped(id int, tenantID *int, userID int, scope NotificationScope) (bool, error) {
	where, args, err := actionPredicate(scope, tenantID, userID, 1)
	if err != nil {
		return false, err
	}
	query := `UPDATE notifications AS n
		SET seen_at = COALESCE(seen_at, NOW())
		WHERE n.id = $1 AND ` + where
	args = append([]interface{}{id}, args...)
	res, err := r.DB.Exec(query, args...)
	if err != nil {
		return false, err
	}
	return rowsAffected(res)
}

// MarkAllSeenScoped marks all of the caller's own unseen notifications seen.
func (r *NotificationRepo) MarkAllSeenScoped(tenantID *int, userID int, scope NotificationScope) error {
	where, args, err := actionPredicate(scope, tenantID, userID, 0)
	if err != nil {
		return err
	}
	_, err = r.DB.Exec(`UPDATE notifications AS n
		SET seen_at = COALESCE(seen_at, NOW())
		WHERE n.seen_at IS NULL AND `+where, args...)
	return err
}

// DeleteScoped removes one of the caller's own notifications.
func (r *NotificationRepo) DeleteScoped(id int, tenantID *int, userID int, scope NotificationScope) (bool, error) {
	where, args, err := actionPredicate(scope, tenantID, userID, 1)
	if err != nil {
		return false, err
	}
	query := `DELETE FROM notifications AS n WHERE n.id = $1 AND ` + where
	args = append([]interface{}{id}, args...)
	res, err := r.DB.Exec(query, args...)
	if err != nil {
		return false, err
	}
	return rowsAffected(res)
}

func (r *NotificationRepo) Delete(id, tenantID, userID int) (bool, error) {
	return r.DeleteScoped(id, &tenantID, userID, ScopeOwn)
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

// Counts returns both badge counts for a viewer. Unread drives the Unread tab;
// unseen drives the bell badge.
func (r *NotificationRepo) Counts(tenantID *int, userID int, scope NotificationScope) (NotificationCounts, error) {
	where, args, err := scopePredicate(scope, tenantID, userID, 0)
	if err != nil {
		return NotificationCounts{}, err
	}
	var counts NotificationCounts
	err = r.DB.QueryRow(`SELECT
			COUNT(*) FILTER (WHERE n.read_at IS NULL),
			COUNT(*) FILTER (WHERE n.seen_at IS NULL)
		FROM notifications n WHERE `+where, args...).Scan(&counts.Unread, &counts.Unseen)
	if err != nil {
		return NotificationCounts{}, err
	}
	return counts, nil
}

// UnreadCount is the legacy single-count wrapper.
func (r *NotificationRepo) UnreadCount(tenantID int, userID *int, scope NotificationScope) (int, error) {
	uid := -1
	if userID != nil {
		uid = *userID
	}
	counts, err := r.Counts(&tenantID, uid, scope)
	return counts.Unread, err
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
