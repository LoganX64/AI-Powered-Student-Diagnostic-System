package repository

import (
	"testing"

	_ "github.com/lib/pq"
)

func TestSubscriptionRepo(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	subRepo := NewSubscriptionRepo(db)

	var tid int
	if err := db.QueryRow(`SELECT id FROM tenants LIMIT 1`).Scan(&tid); err != nil {
		t.Skip("no tenant to test against:", err)
	}

	orig, err := subRepo.GetByTenantID(tid)
	if err != nil {
		t.Fatalf("GetByTenantID: %v", err)
	}
	if orig.PlanName == "" {
		t.Fatalf("expected a plan name for tenant %d", tid)
	}
	// restore everything we mutate at the end
	defer func() {
		subRepo.Upsert(tid, orig.PlanID)
		subRepo.UpdateStatus(tid, orig.Status)
		subRepo.UpdateStorageUsage(tid, orig.StorageUsedBytes)
	}()

	// Upsert to enterprise (id 4) and verify
	if err := subRepo.Upsert(tid, 4); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	upd, err := subRepo.GetByTenantID(tid)
	if err != nil {
		t.Fatalf("GetByTenantID after upsert: %v", err)
	}
	if upd.PlanID != 4 {
		t.Fatalf("Upsert did not change plan_id, got %d", upd.PlanID)
	}

	// UpdateStorageUsage
	if err := subRepo.UpdateStorageUsage(tid, 999); err != nil {
		t.Fatalf("UpdateStorageUsage: %v", err)
	}
	s2, _ := subRepo.GetByTenantID(tid)
	if s2.StorageUsedBytes != 999 {
		t.Fatalf("storage not updated, got %d", s2.StorageUsedBytes)
	}

	// UpdateStatus
	if err := subRepo.UpdateStatus(tid, "cancelled"); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}
	s3, _ := subRepo.GetByTenantID(tid)
	if s3.Status != "cancelled" {
		t.Fatalf("status not updated, got %s", s3.Status)
	}

	// GetUsage mirrors GetByTenantID
	u, err := subRepo.GetUsage(tid)
	if err != nil || u == nil {
		t.Fatalf("GetUsage: %v", err)
	}
}
