package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersionVariables(t *testing.T) {
	// Default values when not set via ldflags
	assert.Equal(t, "dev", Version)
	assert.Equal(t, "none", Commit)
	assert.Equal(t, "unknown", BuildDate)
}

func TestGlobalFlags(t *testing.T) {
	// Test that global flag getters work
	// Note: These test the default values
	assert.False(t, GetVerbose())
	assert.False(t, GetJSONOutput())
}
