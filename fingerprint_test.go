package priv

import (
	"github.com/acoustid/go-acoustid/chromaprint"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"path"
	"testing"
	"time"
)

func loadTestFingerprint(t *testing.T, name string) *chromaprint.Fingerprint {
	data, err := ioutil.ReadFile(path.Join("test_data", name+".txt"))
	require.NoError(t, err)
	fp, err := chromaprint.ParseFingerprintString(string(data))
	require.NoError(t, err)
	return fp
}

func TestMatchFingerprints_NoMatch(t *testing.T) {
	master := loadTestFingerprint(t, "calibre_sunrise")
	query := loadTestFingerprint(t, "radio1_1_ad")
	result, err := MatchFingerprints(master, query)
	if assert.NoError(t, err) {
		assert.Empty(t, result.Sections)
		assert.Equal(t, time.Duration(0), result.MatchingDuration())
	}
}

func TestMatchFingerprints_PartialMatch(t *testing.T) {
	master := loadTestFingerprint(t, "calibre_sunrise")
	query := loadTestFingerprint(t, "radio1_2_ad_and_calibre_sunshine")
	result, err := MatchFingerprints(master, query)
	if assert.NoError(t, err) {
		assert.NotEmpty(t, result.Sections)
		assert.Equal(t, "12.876237s", result.MatchingDuration().String())
	}
}

func TestMatchFingerprints_FullMatch1(t *testing.T) {
	master := loadTestFingerprint(t, "calibre_sunrise")
	query := loadTestFingerprint(t, "radio1_3_calibre_sunshine")
	result, err := MatchFingerprints(master, query)
	if assert.NoError(t, err) {
		assert.NotEmpty(t, result.Sections)
		assert.Equal(t, "17.580979s", result.MatchingDuration().String())
	}
}

func TestMatchFingerprints_FullMatch2(t *testing.T) {
	master := loadTestFingerprint(t, "calibre_sunrise")
	query := loadTestFingerprint(t, "radio1_4_calibre_sunshine")
	result, err := MatchFingerprints(master, query)
	if assert.NoError(t, err) {
		assert.NotEmpty(t, result.Sections)
		assert.Equal(t, "17.580979s", result.MatchingDuration().String())
	}
}

func TestMatchFingerprints_FullMatch3(t *testing.T) {
	master := loadTestFingerprint(t, "calibre_sunrise")
	query := loadTestFingerprint(t, "radio1_5_calibre_sunshine")
	result, err := MatchFingerprints(master, query)
	if assert.NoError(t, err) {
		assert.NotEmpty(t, result.Sections)
		assert.Equal(t, "17.580979s", result.MatchingDuration().String())
	}
}

func TestBoxFilter(t *testing.T) {
	type TestCase struct {
		W      int
		Input  []float64
		Output []float64
	}
	tests := []TestCase{
		{W: 0, Input: []float64{0}, Output: []float64{0}},
		{W: 0, Input: []float64{1, 2}, Output: []float64{0, 0}},
		{W: 1, Input: []float64{1, 2}, Output: []float64{1, 2}},
		{W: 2, Input: []float64{1, 2}, Output: []float64{(1 + 2) / 2.0, (1 + 2) / 2.0}},
		{W: 3, Input: []float64{1, 2}, Output: []float64{(2 + 1 + 2) / 3.0, (1 + 2 + 1) / 3.0}},
		{W: 4, Input: []float64{1, 2}, Output: []float64{(1 + 2) / 2.0, (1 + 2) / 2.0}},
		{W: 8, Input: []float64{1, 2}, Output: []float64{(1 + 2) / 2.0, (1 + 2) / 2.0}},
		{W: 3, Input: []float64{1, 2, 3, 4, 5}, Output: []float64{(2 + 1 + 2) / 3.0, (1 + 2 + 3) / 3.0, (2 + 3 + 4) / 3.0, (3 + 4 + 5) / 3.0, (4 + 5 + 4) / 3.0}},
	}
	for _, test := range tests {
		tmp := make([]float64, len(test.Input))
		boxFilter(test.Input, tmp, test.W)
		assert.Equal(t, test.Output, tmp)
	}
}

func TestGaussianFilter(t *testing.T) {
	type TestCase struct {
		N      int
		Sigma  float64
		Input  []float64
		Output []float64
	}
	tests := []TestCase{
		{N: 2, Sigma: 3.6, Input: []float64{1, 2, 3, 4, 5}, Output: []float64{3.2222222222222223, 3.111111111111111, 3, 2.888888888888889, 2.7777777777777777}},
		{N: 3, Sigma: 3.6, Input: []float64{1, 2, 3, 4, 5}, Output: []float64{2.7142857142857144, 2.857142857142857, 3, 3.142857142857143, 3.2857142857142856}},
		{N: 4, Sigma: 3.6, Input: []float64{1, 2, 3, 4, 5}, Output: []float64{2.2, 2.4, 3, 3.6, 3.8}},
		{N: 5, Sigma: 3.6, Input: []float64{1, 2, 3, 4, 5}, Output: []float64{2.2, 2.4, 3, 3.6, 3.8}},
	}
	for _, test := range tests {
		tmp := make([]float64, len(test.Input))
		gaussianFilter(test.Input, tmp, test.Sigma, test.N)
		assert.Equal(t, test.Output, tmp)
	}
}

func TestGradient(t *testing.T) {
	type TestCase struct {
		N      int
		Input  []float64
		Output []float64
	}
	tests := []TestCase{
		{N: 0, Input: []float64{0}, Output: []float64{0}},
		{N: 0, Input: []float64{1, 2}, Output: []float64{0, 0}},
		{N: 1, Input: []float64{1, 2}, Output: []float64{1, 1}},
		{N: 2, Input: []float64{1, 2}, Output: []float64{1, 1}},
		{N: 3, Input: []float64{1, 2}, Output: []float64{1, 1}},
		{N: 4, Input: []float64{1, 2}, Output: []float64{1, 1}},
		{N: 8, Input: []float64{1, 2}, Output: []float64{1, 1}},
		{N: 1, Input: []float64{1, 2, 4, 8, 16}, Output: []float64{2 - 1, 4 - 2, 8 - 4, 16 - 8, 16 - 8}},
		{N: 2, Input: []float64{1, 2, 4, 8, 16}, Output: []float64{4 - 1, 4 - 1, 8 - 2, 16 - 4, 16 - 4}},
		{N: 3, Input: []float64{1, 2, 4, 8, 16}, Output: []float64{8 - 1, 8 - 1, 16 - 2, 16 - 2, 16 - 2}},
	}
	for _, test := range tests {
		tmp := make([]float64, len(test.Input))
		gradient(test.Input, tmp, test.N)
		assert.Equal(t, test.Output, tmp)
	}
}
