package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newTestManager splits a temp dir into built-in and custom maps so the
// ownership rules can be exercised in isolation.
func newTestManager(t *testing.T) (*Manager, string) {
	t.Helper()
	root := createTestConfigDir(t)
	t.Cleanup(func() { os.RemoveAll(root) })

	builtinDir := filepath.Join(root, "builtin")
	if err := os.MkdirAll(builtinDir, 0755); err != nil {
		t.Fatalf("mkdir builtin: %v", err)
	}
	writeConfigFile(t, builtinDir, "classic", createValidConfig())

	t.Setenv("CUSTOM_MAPS_DIR", filepath.Join(root, "custom"))
	m, err := NewManager(builtinDir)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	return m, m.CustomDir()
}

func TestCreateCustomReturnsUsableEditToken(t *testing.T) {
	m, customDir := newTestManager(t)

	cfg := createValidConfig()
	cfg.Name = "mine"

	token, err := m.CreateCustom("mine", cfg)
	if err != nil {
		t.Fatalf("CreateCustom: %v", err)
	}
	if token == "" {
		t.Fatal("expected a non-empty edit token")
	}
	if strings.Contains(token, "=") {
		t.Errorf("token should be URL-safe without padding, got %q", token)
	}

	// The map file and token sidecar both land in the custom directory.
	if _, err := os.Stat(filepath.Join(customDir, "mine.json")); err != nil {
		t.Errorf("expected map file in custom dir: %v", err)
	}
	tokenPath := filepath.Join(customDir, "mine.json"+editTokenSuffix)
	raw, err := os.ReadFile(tokenPath)
	if err != nil {
		t.Fatalf("expected token sidecar: %v", err)
	}
	if strings.Contains(string(raw), token) {
		t.Error("token must be stored hashed, not in plaintext")
	}

	if err := m.VerifyEditToken("mine", token); err != nil {
		t.Errorf("VerifyEditToken with correct token: %v", err)
	}
	if err := m.VerifyEditToken("mine", "wrong"); !errors.Is(err, ErrInvalidEditToken) {
		t.Errorf("expected ErrInvalidEditToken, got %v", err)
	}
	if err := m.VerifyEditToken("mine", ""); !errors.Is(err, ErrEditTokenRequired) {
		t.Errorf("expected ErrEditTokenRequired, got %v", err)
	}
}

func TestCreateCustomNeverOverwrites(t *testing.T) {
	m, _ := newTestManager(t)

	// Built-in names are taken even though they live in the other directory.
	if _, err := m.CreateCustom("classic", createValidConfig()); !errors.Is(err, ErrMapExists) {
		t.Fatalf("expected ErrMapExists for built-in name, got %v", err)
	}

	cfg := createValidConfig()
	if _, err := m.CreateCustom("mine", cfg); err != nil {
		t.Fatalf("first create: %v", err)
	}
	if _, err := m.CreateCustom("mine", cfg); !errors.Is(err, ErrMapExists) {
		t.Fatalf("expected ErrMapExists on duplicate, got %v", err)
	}
}

func TestBuiltinMapsAreImmutable(t *testing.T) {
	m, _ := newTestManager(t)

	if !m.IsBuiltin("classic") {
		t.Fatal("classic should be reported as built-in")
	}

	if err := m.SaveConfig("classic", createValidConfig()); !errors.Is(err, ErrReadOnlyMap) {
		t.Errorf("SaveConfig on built-in: expected ErrReadOnlyMap, got %v", err)
	}
	if err := m.DeleteConfig("classic"); !errors.Is(err, ErrReadOnlyMap) {
		t.Errorf("DeleteConfig on built-in: expected ErrReadOnlyMap, got %v", err)
	}

	// A token can never unlock a built-in map, even one that looks valid.
	if err := m.VerifyEditToken("classic", "anything"); !errors.Is(err, ErrReadOnlyMap) {
		t.Errorf("VerifyEditToken on built-in: expected ErrReadOnlyMap, got %v", err)
	}

	// And the shipped file is still loadable.
	if _, err := m.LoadConfig("classic"); err != nil {
		t.Errorf("built-in map should still load: %v", err)
	}
}

func TestCustomMapUpdateAndDelete(t *testing.T) {
	m, _ := newTestManager(t)

	cfg := createValidConfig()
	cfg.Name = "mine"
	if _, err := m.CreateCustom("mine", cfg); err != nil {
		t.Fatalf("CreateCustom: %v", err)
	}

	updated := createValidConfig()
	updated.Name = "mine renamed"
	updated.MaxBattery = 12
	updated.StartingBattery = 10
	if err := m.SaveConfig("mine", updated); err != nil {
		t.Fatalf("SaveConfig on custom map: %v", err)
	}
	loaded, err := m.LoadConfig("mine")
	if err != nil {
		t.Fatalf("LoadConfig after update: %v", err)
	}
	if loaded.Name != "mine renamed" {
		t.Errorf("update not persisted, got name %q", loaded.Name)
	}

	if err := m.DeleteConfig("mine"); err != nil {
		t.Fatalf("DeleteConfig: %v", err)
	}
	if _, err := m.LoadConfig("mine"); !errors.Is(err, ErrConfigNotFound) {
		t.Errorf("expected map to be gone, got %v", err)
	}
	if err := m.VerifyEditToken("mine", "test"); !errors.Is(err, ErrConfigNotFound) {
		t.Errorf("expected ErrConfigNotFound after delete, got %v", err)
	}
}

func TestListConfigsReportsOwnership(t *testing.T) {
	m, _ := newTestManager(t)

	cfg := createValidConfig()
	cfg.Name = "mine"
	if _, err := m.CreateCustom("mine", cfg); err != nil {
		t.Fatalf("CreateCustom: %v", err)
	}

	infos, err := m.ListConfigs()
	if err != nil {
		t.Fatalf("ListConfigs: %v", err)
	}

	got := make(map[string]bool, len(infos))
	for _, info := range infos {
		got[info.MapID] = info.Builtin
	}
	if builtin, ok := got["classic"]; !ok || !builtin {
		t.Errorf("classic should be listed as built-in, got %v (present=%v)", builtin, ok)
	}
	if builtin, ok := got["mine"]; !ok || builtin {
		t.Errorf("mine should be listed as user-created, got %v (present=%v)", builtin, ok)
	}
}

func TestCustomMapDoesNotShadowBuiltinOnDisk(t *testing.T) {
	m, customDir := newTestManager(t)

	// Simulate a stray file that predates the ownership split.
	if err := os.WriteFile(filepath.Join(customDir, "classic.json"), []byte(`{"name":"impostor"}`), 0644); err != nil {
		t.Fatalf("write stray file: %v", err)
	}

	if !m.IsBuiltin("classic") {
		t.Error("classic should still be built-in")
	}

	infos, err := m.ListConfigs()
	if err != nil {
		t.Fatalf("ListConfigs: %v", err)
	}
	seen := 0
	for _, info := range infos {
		if info.MapID != "classic" {
			continue
		}
		seen++
		if !info.Builtin {
			t.Error("shadowing custom file must not make classic look user-created")
		}
	}
	if seen != 1 {
		t.Errorf("expected exactly one classic entry, got %d", seen)
	}
}
