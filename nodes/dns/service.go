package dns

import "context"

// Service is the expected interface to interface with a DNS provider.
type Service interface {
	// List returns all records of domain.
	List(ctx context.Context, domain string) ([]Record, error)
	// Create creates records in domain.
	// If a record has a TTL of zero or less the implementation should
	// fall back to the default the DNS provider expects.
	Create(ctx context.Context, domain string, records []Record) error
	// Update replaces existing records in domain.
	// If a record has a TTL of zero or less the implementation should
	// fall back to the default the DNS provider expects.
	Update(ctx context.Context, domain string, records []Record) error
	// Delete removes records from domain.
	Delete(ctx context.Context, domain string, records []Record) error
}
