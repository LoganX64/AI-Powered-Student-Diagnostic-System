package repository

import (
	"testing"

	"ai-student-diagnostic/backend/internal/testutil"
)

// TestTenantRepoListPlanFilterIsBound pins the SQL-injection fix at the layer the
// defect actually lived in, independently of any HTTP path.
//
// TenantRepo.List used to build its plan clause by concatenation:
//
//	"AND ts.plan_id = (SELECT id FROM subscription_plans WHERE slug = '" + planFilter + "')"
//
// so a payload that closed the literal and appended its own OR returned every
// tenant on the targeted plan. The clause is now bound, so such a payload is
// compared as a literal slug and matches nothing.
//
// List is a global (super-admin) listing, so it also returns dev tenants and the
// counts depend on whatever is in the database. Every assertion below is
// therefore either about our own tenant ids or a delta against a measured
// baseline — never an absolute total.
func TestTenantRepoListPlanFilterIsBound(t *testing.T) {
	db := testutil.OpenTestDB(t)
	r := NewTenantRepo(db)

	// subscribed goes on a real plan; unsubscribed stays on none, which the
	// LEFT JOIN reports as NULL plan_id and the "free" branch counts as free.
	subscribed := testutil.CreateTenant(t, db)
	unsubscribed := testutil.CreateTenant(t, db)

	var planID int
	if err := db.QueryRow(`SELECT id FROM subscription_plans WHERE slug = 'professional'`).Scan(&planID); err != nil {
		t.Skipf("professional plan not seeded: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO tenant_subscriptions (tenant_id, plan_id) VALUES ($1, $2)`, subscribed, planID); err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	ids := func(rows []TenantRow) map[int]bool {
		out := map[int]bool{}
		for _, row := range rows {
			out[row.ID] = true
		}
		return out
	}

	// baseline is the unfiltered result set: every plan-filtered result must be a
	// subset of it, so the deltas below are exact regardless of dev data.
	baseRows, _, err := r.List("", "", 500, 0)
	if err != nil {
		t.Fatalf("unfiltered List: %v", err)
	}
	base := ids(baseRows)
	if !base[subscribed] || !base[unsubscribed] {
		t.Fatalf("fixture tenants missing from the unfiltered listing: %d=%v %d=%v",
			subscribed, base[subscribed], unsubscribed, base[unsubscribed])
	}

	// Asserting total == len(rows) would be wrong here: other packages in this
	// test binary create fixture tenants in parallel with this one, so `total` is
	// a moving target. Membership of our own two tenants is the stable fact.
	t.Run("no filter returns both", func(t *testing.T) {
		rows, _, err := r.List("", "", 500, 0)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		got := ids(rows)
		if !got[subscribed] || !got[unsubscribed] {
			t.Fatalf("both fixture tenants must appear unfiltered: %+v", got)
		}
	})

	t.Run("real slug matches only the subscribed tenant", func(t *testing.T) {
		rows, _, err := r.List("", "professional", 500, 0)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		got := ids(rows)
		if !got[subscribed] {
			t.Fatalf("professional filter excluded tenant %d: %+v", subscribed, got)
		}
		if got[unsubscribed] {
			t.Fatalf("professional filter wrongly included the unsubscribed tenant %d", unsubscribed)
		}
	})

	t.Run("free matches the unsubscribed tenant", func(t *testing.T) {
		rows, _, err := r.List("", "free", 500, 0)
		if err != nil {
			t.Fatalf("free: %v", err)
		}
		got := ids(rows)
		if !got[unsubscribed] {
			t.Fatalf("free filter excluded tenant %d (NULL plan_id counts as free)", unsubscribed)
		}
		if got[subscribed] {
			t.Fatalf("free filter wrongly included the subscribed tenant %d", subscribed)
		}
	})

	t.Run("paid excludes the free tenant", func(t *testing.T) {
		rows, _, err := r.List("", "paid", 500, 0)
		if err != nil {
			t.Fatalf("paid: %v", err)
		}
		got := ids(rows)
		if got[unsubscribed] {
			t.Fatalf("paid filter wrongly included the free tenant %d", unsubscribed)
		}
		if !got[subscribed] {
			t.Fatalf("paid filter excluded the professional tenant %d", subscribed)
		}
	})

	t.Run("unknown slug matches nothing", func(t *testing.T) {
		rows, _, err := r.List("", "nonexistent", 500, 0)
		if err != nil {
			t.Fatalf("unknown slug: %v", err)
		}
		if len(rows) != 0 {
			t.Fatalf("unknown slug returned %d rows, want 0", len(rows))
		}
	})

	// Each payload closes the literal and appends its own boolean. A successful
	// injection would widen the result set beyond the baseline; an error means the
	// payload instead broke the SQL, which is also not a leak.
	for _, payload := range []string{
		"x' OR slug='professional' OR slug='x",
		"x' OR 1=1 OR slug='x",
		"x' OR ts.plan_id IS NOT NULL OR slug='x",
		"' OR '1'='1",
		"x' UNION SELECT 1--",
	} {
		t.Run("payload "+payload, func(t *testing.T) {
			// A bound parameter compares the payload as a slug that does not exist,
			// so the filter must be empty. This is stable under concurrent fixture
			// creation: no tenant can be on a plan literally named after the payload.
			rows, total, err := r.List("", payload, 500, 0)
			if err != nil {
				// An error means the payload broke the SQL rather than being
				// compared as a literal. That is not a leak either.
				t.Logf("payload produced an error rather than a match: %v", err)
				return
			}
			if len(rows) != 0 || total != 0 {
				t.Fatalf("STILL INJECTABLE: payload returned total=%d rows=%d, which can only happen if the value reached the SQL as text", total, len(rows))
			}
		})
	}

	t.Run("search and plan together", func(t *testing.T) {
		// A bound plan param shifts LIMIT/OFFSET to $3/$4, so a mis-numbered
		// placeholder fails here.
		rows, _, err := r.List("fixtures", "professional", 500, 0)
		if err != nil {
			t.Fatalf("search+plan: %v", err)
		}
		if !ids(rows)[subscribed] {
			t.Fatalf("search+plan excluded tenant %d: %+v", subscribed, rows)
		}
		if _, _, err := r.List("fixtures", "professional", 1, 1); err != nil {
			t.Fatalf("paged search+plan: %v", err)
		}
		if _, _, err := r.List("fixtures", "", 1, 1); err != nil {
			t.Fatalf("paged search only: %v", err)
		}
	})
}