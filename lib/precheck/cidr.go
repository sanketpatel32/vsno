package precheck

import (
	"fmt"
	"net/netip"
)

// Overlaps reports whether two CIDRs share any address.
func Overlaps(a, b string) (bool, error) {
	pa, err := netip.ParsePrefix(a)
	if err != nil {
		return false, fmt.Errorf("parsing %q: %w", a, err)
	}
	pb, err := netip.ParsePrefix(b)
	if err != nil {
		return false, fmt.Errorf("parsing %q: %w", b, err)
	}
	return pa.Overlaps(pb), nil
}

// CIDRConflict describes a guest network that collides with a host network.
type CIDRConflict struct {
	Guest, Host string
}

// FindCIDRConflicts returns every guest/host pair that overlaps. Invalid CIDRs
// are returned as an error.
func FindCIDRConflicts(guest, host []string) ([]CIDRConflict, error) {
	var out []CIDRConflict
	for _, g := range guest {
		for _, h := range host {
			ov, err := Overlaps(g, h)
			if err != nil {
				return nil, err
			}
			if ov {
				out = append(out, CIDRConflict{Guest: g, Host: h})
			}
		}
	}
	return out, nil
}
