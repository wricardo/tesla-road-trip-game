package session

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/wricardo/tesla-road-trip-game/game/config"
)

// Reproduces production: many persisted sessions on maps with many parks.
// Rehydration must not re-run the winnability proof.
func TestRehydrateManySessionsIsCheap(t *testing.T) {
	dir := t.TempDir()
	maps := filepath.Join(dir, "maps")
	if err := os.MkdirAll(maps, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, m := range []string{"classic", "stella_map", "sarah_map"} {
		data, err := os.ReadFile(filepath.Join("../../maps", m+".json"))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(maps, m+".json"), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cfgMgr, err := config.NewManager(maps)
	if err != nil {
		t.Fatal(err)
	}

	saveDir := filepath.Join(dir, "sessions")
	fp, err := NewFilePersistence(saveDir, cfgMgr)
	if err != nil {
		t.Fatal(err)
	}
	mgr := NewManagerWithPersistence(fp)

	names := []string{"classic", "stella_map", "sarah_map"}
	const n = 78
	for i := 0; i < n; i++ {
		name := names[i%len(names)]
		cfg, err := cfgMgr.LoadConfig(name)
		if err != nil {
			t.Fatal(err)
		}
		sess, err := mgr.Create(fmt.Sprintf("s%03d", i), cfg)
		if err != nil {
			t.Fatal(err)
		}
		if err := mgr.Save(sess.ID); err != nil {
			t.Fatal(err)
		}
	}

	// Fresh manager: the one above already holds these sessions, and
	// LoadPersistedSessions skips anything already in memory.
	fresh := NewManagerWithPersistence(fp)

	var before runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)

	start := time.Now()
	if err := fresh.LoadPersistedSessions(); err != nil {
		t.Fatal(err)
	}
	elapsed := time.Since(start)

	var after runtime.MemStats
	runtime.ReadMemStats(&after)
	allocMB := float64(after.TotalAlloc-before.TotalAlloc) / (1 << 20)

	loaded := len(fresh.List())
	t.Logf("rehydrated %d sessions in %v (allocated %.1f MB)", loaded, elapsed, allocMB)
	if loaded != n {
		t.Fatalf("expected %d sessions to rehydrate, got %d", n, loaded)
	}
	if elapsed > 5*time.Second {
		t.Fatalf("rehydration too slow: %v", elapsed)
	}
	if allocMB > 200 {
		t.Fatalf("rehydration allocated too much: %.1f MB", allocMB)
	}
}
