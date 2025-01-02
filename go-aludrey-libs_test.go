package goaludreylibs

import (
	"testing"
)

func TestVersion(t *testing.T) {
	version := Version()
	if version == "" {
		t.Errorf("Error: %v", version)
	}
}
