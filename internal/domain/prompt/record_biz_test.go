package prompt

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"prompter/infra"
)

// strPtr returns a pointer to the given string, useful for setting
// optional CustomText fields in test data.
func strPtr(s string) *string {
	return &s
}

// TestPersistFromActive_DraftRepoWithMiniredis validates the DraftRepo
// Redis layer used by RecordBiz.PersistFromActive. Since SetActive
// uses PutObject which calls SetEx with 0 expiration (invalid Redis),
// we seed data directly via redis SET and verify GetActive reads it
// correctly. GetActive is the read path that PersistFromActive uses
// to retrieve the active prompt from Redis.
func TestPersistFromActive_DraftRepoWithMiniredis(t *testing.T) {
	// Setup: start a miniredis server
	s, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	// Wrap the miniredis-backed redis.Client into infra.RedisClient
	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	rc := infra.NewCustomRedisClient(rdb)

	// Build Data with only the RedisClient populated (DB is nil —
	// DraftRepo does not use DB, and RecordBiz/SliceRepo are not
	// exercised here because they require a real *gorm.DB).
	data := &infra.Data{RedisClient: rc}
	repo := NewDraftRepo(data)

	ctx := context.Background()
	key := activePromptKey

	// --- Subtest 1: GetActive on empty Redis returns nil ---
	t.Run("GetActive_Empty_Returns_Nil", func(t *testing.T) {
		active, err := repo.GetActive()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if active != nil {
			t.Fatal("expected nil for empty active prompt")
		}
	})

	// --- Subtest 2: GetActive after writing valid JSON ---
	t.Run("GetActive_Reads_Stored_JSON", func(t *testing.T) {
		input := ActivePromptData{
			Title: "test title",
			Regions: []ActivePromptRegionDTO{
				{
					RegionID: 1, RegionName: "人物", SortOrder: 0,
					Slices: []ActiveSliceDTO{
						{SliceID: 5, CustomText: nil, SortOrder: 0},
						{SliceID: 12, CustomText: strPtr("custom"), SortOrder: 1},
					},
				},
			},
			UpdatedAt: "2024-01-01T00:00:00Z",
		}
		raw, err := json.Marshal(input)
		if err != nil {
			t.Fatalf("json.Marshal failed: %v", err)
		}

		// Write directly via redis SET (bypasses SetEx-with-0 bug in PutObject)
		if err := rdb.Set(ctx, key, string(raw), 0).Err(); err != nil {
			t.Fatalf("redis SET failed: %v", err)
		}

		got, err := repo.GetActive()
		if err != nil {
			t.Fatalf("GetActive failed: %v", err)
		}
		if got == nil {
			t.Fatal("expected non-nil after writing data to Redis")
		}

		// Title
		if got.Title != "test title" {
			t.Errorf("title: got %q want %q", got.Title, "test title")
		}

		// UpdatedAt
		if got.UpdatedAt != "2024-01-01T00:00:00Z" {
			t.Errorf("UpdatedAt: got %q want %q", got.UpdatedAt, "2024-01-01T00:00:00Z")
		}

		// Regions count
		if len(got.Regions) != 1 {
			t.Fatalf("regions length: got %d want 1", len(got.Regions))
		}
		r := got.Regions[0]
		if r.RegionID != 1 || r.RegionName != "人物" {
			t.Errorf("region: got id=%d name=%q want id=1 name=人物", r.RegionID, r.RegionName)
		}
		if len(r.Slices) != 2 {
			t.Fatalf("slices length: got %d want 2", len(r.Slices))
		}

		// Slice 0 — nil CustomText
		if r.Slices[0].SliceID != 5 {
			t.Errorf("slice[0].SliceID: got %d want 5", r.Slices[0].SliceID)
		}
		if r.Slices[0].CustomText != nil {
			t.Error("slice[0].CustomText: expected nil")
		}

		// Slice 1 — non-nil CustomText
		if r.Slices[1].SliceID != 12 {
			t.Errorf("slice[1].SliceID: got %d want 12", r.Slices[1].SliceID)
		}
		if r.Slices[1].CustomText == nil {
			t.Fatal("slice[1].CustomText: expected non-nil")
		}
		if *r.Slices[1].CustomText != "custom" {
			t.Errorf("slice[1].CustomText: got %q want %q", *r.Slices[1].CustomText, "custom")
		}
		if r.Slices[1].SortOrder != 1 {
			t.Errorf("slice[1].SortOrder: got %d want 1", r.Slices[1].SortOrder)
		}
	})

	// --- Subtest 3: GetActive returns nil for invalid JSON (Redis corrupt, MySQL unavailable) ---
	t.Run("GetActive_Invalid_JSON_Fallback_Nil", func(t *testing.T) {
		if err := rdb.Set(ctx, key, "not-valid-json", 0).Err(); err != nil {
			t.Fatalf("redis SET failed: %v", err)
		}

		got, err := repo.GetActive()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != nil {
			t.Fatal("expected nil when Redis has invalid JSON and DB is unavailable")
		}
	})

	// --- Subtest 4: Overwrite in Redis, GetActive reflects latest ---
	t.Run("GetActive_Reflects_Overwrite", func(t *testing.T) {
		first := ActivePromptData{
			Title: "first title",
			Regions: []ActivePromptRegionDTO{
				{RegionID: 1, RegionName: "人物", SortOrder: 0,
					Slices: []ActiveSliceDTO{{SliceID: 1, SortOrder: 0}}},
			},
		}
		raw1, _ := json.Marshal(first)
		if err := rdb.Set(ctx, key, string(raw1), 0).Err(); err != nil {
			t.Fatalf("redis SET first failed: %v", err)
		}

		second := ActivePromptData{
			Title: "second title",
			Regions: []ActivePromptRegionDTO{
				{RegionID: 3, RegionName: "武器", SortOrder: 0,
					Slices: []ActiveSliceDTO{
						{SliceID: 7, SortOrder: 0},
						{SliceID: 8, SortOrder: 1},
					}},
			},
		}
		raw2, _ := json.Marshal(second)
		if err := rdb.Set(ctx, key, string(raw2), 0).Err(); err != nil {
			t.Fatalf("redis SET second failed: %v", err)
		}

		got, err := repo.GetActive()
		if err != nil {
			t.Fatalf("GetActive after overwrite failed: %v", err)
		}

		if got.Title != "second title" {
			t.Errorf("title after overwrite: got %q want %q", got.Title, "second title")
		}
		if len(got.Regions) != 1 || len(got.Regions[0].Slices) != 2 {
			t.Fatalf("regions/slices after overwrite: got %d regions, %d slices", len(got.Regions), len(got.Regions[0].Slices))
		}
		if got.Regions[0].Slices[0].SliceID != 7 {
			t.Errorf("slice[0].SliceID: got %d want 7", got.Regions[0].Slices[0].SliceID)
		}
	})
}
