package priv

import (
	"io/ioutil"
	"path"
	"testing"

	"github.com/acoustid/go-acoustid/chromaprint"
	"github.com/stretchr/testify/require"
)

func loadTestFingerprint(t *testing.T, name string) *chromaprint.Fingerprint {
	data, err := ioutil.ReadFile(path.Join("test_data", name+".txt"))
	require.NoError(t, err)
	fp, err := chromaprint.ParseFingerprintString(string(data))
	require.NoError(t, err)
	return fp
}
