package priv

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestIsValidCatalogName(t *testing.T) {
	assert.True(t, IsValidCatalogName("test"))
	assert.False(t, IsValidCatalogName("_test"))
}

func TestIsValidTrackID(t *testing.T) {
	assert.True(t, IsValidTrackID("test"))
	assert.False(t, IsValidTrackID("_test"))
}
