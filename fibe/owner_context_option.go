package fibe

import (
	"fmt"
	"net/http"
	"strconv"
)

// WithOwnerContext verifies that the chosen credential belongs to its profile.
// The server remains authoritative; these headers never grant a team context.
func WithOwnerContext(owner OwnerContext) Option {
	return func(cfg *clientConfig) {
		cfg.expectedOwner = &owner
		previousRequest, previousResponse := cfg.requestHook, cfg.responseHook
		cfg.requestHook = func(req *http.Request) error {
			if owner.OwnerID <= 0 || (owner.OwnerType != "Player" && owner.OwnerType != "Team") {
				return fmt.Errorf("invalid owner context")
			}
			if previousRequest != nil {
				if err := previousRequest(req); err != nil {
					return err
				}
			}
			req.Header.Set("X-Fibe-Expected-Owner-Type", owner.OwnerType)
			req.Header.Set("X-Fibe-Expected-Owner-ID", strconv.FormatInt(owner.OwnerID, 10))
			return nil
		}
		cfg.responseHook = func(res *http.Response) error {
			if res.StatusCode >= 200 && res.StatusCode < 300 && (res.Header.Get("X-Fibe-Owner-Type") != owner.OwnerType ||
				res.Header.Get("X-Fibe-Owner-ID") != strconv.FormatInt(owner.OwnerID, 10)) {
				return fmt.Errorf("credential response does not match selected owner context")
			}
			if previousResponse != nil {
				return previousResponse(res)
			}
			return nil
		}
	}
}
