package priv

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestIsValidCatalogName(t *testing.T) {
	assert.True(t, IsValidCatalogName("test"))
	assert.False(t, IsValidCatalogName("_test"))
}

func TestIsValidTrackID(t *testing.T) {
	assert.True(t, IsValidTrackID("test"))
	assert.False(t, IsValidTrackID("_test"))
}
