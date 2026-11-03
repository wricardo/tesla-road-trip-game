package config

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/wricardo/tesla-road-trip-game/game/engine"
)

// editTokenSuffix is appended to a custom map's filename to store the hash of
// the edit token handed out at creation time.
const editTokenSuffix = ".edit-token"

var (
	// ErrReadOnlyMap is returned when a caller tries to mutate a built-in map
	// without admin authorization.
	ErrReadOnlyMap = errors.New("built-in map is read-only")
	// ErrEditTokenRequired is returned when a custom map is mutated without a token.
	ErrEditTokenRequired = errors.New("edit token required to modify this map")
	// ErrInvalidEditToken is returned when the supplied token does not match.
	ErrInvalidEditToken = errors.New("invalid edit token")
)

// editTokenRecord is the on-disk sidecar for a custom map.
type editTokenRecord struct {
	TokenHash string `json:"token_hash"`
}

// CreateCustom stores a new user-created map and returns the edit token that
// authorizes future updates and deletes. It never overwrites: if the name is
// taken by a built-in or another custom map, ErrMapExists is returned.
func (m *Manager) CreateCustom(name string, config *engine.GameConfig) (string, error) {
	if err := validateConfigName(name); err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}
	if err := engine.ValidateGameConfig(config); err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.IsBuiltin(name) || m.isCustom(name) {
		return "", fmt.Errorf("%w: %q", ErrMapExists, name)
	}

	token, err := newEditToken()
	if err != nil {
		return "", err
	}
	if err := m.writeEditToken(name, token); err != nil {
		return "", err
	}
	if err := m.writeCustomLocked(name, config); err != nil {
		// Roll back the sidecar so a failed create leaves nothing behind.
		os.Remove(m.editTokenPath(name))
		return "", err
	}
	return token, nil
}

// VerifyEditToken reports whether token authorizes modifying a custom map.
// Built-in maps are never authorized by a token; use admin credentials instead.
func (m *Manager) VerifyEditToken(name, token string) error {
	if m.IsBuiltin(name) {
		return fmt.Errorf("%w: %q", ErrReadOnlyMap, name)
	}
	if !m.isCustom(name) {
		return fmt.Errorf("%w: %q", ErrConfigNotFound, name)
	}
	if strings.TrimSpace(token) == "" {
		return ErrEditTokenRequired
	}

	data, err := os.ReadFile(m.editTokenPath(name))
	if err != nil {
		// A custom map written before token support has no sidecar; treat it as
		// token-less and require admin authorization instead of allowing edits.
		return ErrEditTokenRequired
	}
	var rec editTokenRecord
	if err := json.Unmarshal(data, &rec); err != nil || rec.TokenHash == "" {
		return ErrEditTokenRequired
	}

	got := hashEditToken(token)
	if subtle.ConstantTimeCompare([]byte(got), []byte(rec.TokenHash)) != 1 {
		return ErrInvalidEditToken
	}
	return nil
}

// writeEditToken stores the hash of token next to the custom map file.
func (m *Manager) writeEditToken(name, token string) error {
	rec := editTokenRecord{TokenHash: hashEditToken(token)}
	data, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal edit token record: %w", err)
	}
	if err := os.WriteFile(m.editTokenPath(name), data, 0600); err != nil {
		return fmt.Errorf("failed to write edit token: %w", err)
	}
	return nil
}

func (m *Manager) editTokenPath(name string) string {
	base := strings.TrimSuffix(filepath.Base(strings.TrimSpace(name)), ".json")
	return filepath.Join(m.customDir, base+".json"+editTokenSuffix)
}

// writeCustomLocked persists config without taking m.mu (callers hold it).
func (m *Manager) writeCustomLocked(name string, config *engine.GameConfig) error {
	base := strings.TrimSuffix(name, ".json")
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	if err := os.WriteFile(filepath.Join(m.customDir, base+".json"), data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	m.configs[base] = config
	m.configs[name] = config
	return nil
}

// newEditToken returns a URL-safe random token (256 bits of entropy).
func newEditToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("failed to generate edit token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func hashEditToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
