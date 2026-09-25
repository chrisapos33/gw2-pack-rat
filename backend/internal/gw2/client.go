package gw2

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
)

const baseURL = "https://api.guildwars2.com"

// requiredPermissions is the exact set this tool needs — no more, no less.
var requiredPermissions = []string{"account", "characters", "inventories", "unlocks"}

type tokenInfo struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
}

// ValidationError describes a permission mismatch so callers can surface clear messages.
type ValidationError struct {
	Missing []string
	Extra   []string
}

func (e *ValidationError) Error() string {
	var parts []string
	if len(e.Missing) > 0 {
		parts = append(parts, fmt.Sprintf("missing required permissions: %s", strings.Join(e.Missing, ", ")))
	}
	if len(e.Extra) > 0 {
		parts = append(parts, fmt.Sprintf(
			"key has extra permissions we don't need (%s) — please create a new key with only: %s",
			strings.Join(e.Extra, ", "),
			strings.Join(requiredPermissions, ", "),
		))
	}
	return strings.Join(parts, "; ")
}

// ValidateKey calls /v2/tokeninfo, enforces exactly the required permission set,
// and returns the granted permissions on success.
func ValidateKey(key string) ([]string, error) {
	resp, err := http.Get(fmt.Sprintf("%s/v2/tokeninfo?access_token=%s", baseURL, key))
	if err != nil {
		return nil, fmt.Errorf("failed to reach GW2 API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("invalid API key")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GW2 API returned status %d", resp.StatusCode)
	}

	var info tokenInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to parse GW2 response: %w", err)
	}

	granted := make(map[string]bool, len(info.Permissions))
	for _, p := range info.Permissions {
		granted[p] = true
	}
	required := make(map[string]bool, len(requiredPermissions))
	for _, p := range requiredPermissions {
		required[p] = true
	}

	var missing, extra []string
	for _, p := range requiredPermissions {
		if !granted[p] {
			missing = append(missing, p)
		}
	}
	for _, p := range info.Permissions {
		if !required[p] {
			extra = append(extra, p)
		}
	}

	if len(missing) > 0 || len(extra) > 0 {
		sort.Strings(missing)
		sort.Strings(extra)
		return nil, &ValidationError{Missing: missing, Extra: extra}
	}

	return info.Permissions, nil
}
