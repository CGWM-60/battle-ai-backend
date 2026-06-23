package translations

import (
	"context"

	"gorm.io/gorm"
)

// SeedForcedContentDescriptions used to refresh Nexus/MMO catalogue labels.
// The Nexus/MMO catalogue translation domains are deprecated and purged, so the
// hook remains as a no-op for startup compatibility.
func SeedForcedContentDescriptions(ctx context.Context, database *gorm.DB) error {
	return nil
}
