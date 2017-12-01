package priv

import (
	"github.com/acoustid/go-acoustid/chromaprint"
	"github.com/acoustid/go-acoustid/util"
	"github.com/pkg/errors"
	"math"
	"sort"
	"time"
)

const NumQueryBits = 26
const NumAlignBits = 14
const NumOffsetCandidates = 3
const MaxOffsetThresholdDiv = 10

func hashBitMask(nbits int) uint32 {
	var mask uint32 = 0xaaaaaaaa
	if nbits <= 16 {
		mask &= (1 << (uint(nbits) * 2)) - 1
	} else {
		mask |= 0x55555555 & ((1 << (uint(nbits-16) * 2)) - 1)
	}
	return mask
}

func ExtractQuery(fp *chromaprint.Fingerprint) []int32 {
	mask := hashBitMask(NumQueryBits)
	query := make([]int32, len(fp.Hashes))
	for i := 0; i < len(fp.Hashes); i++ {
		query[i] = int32(fp.Hashes[i] & mask)
	}
	return query
}

type FingerprintConfig struct {
	SampleRate            int
	FrameSize             int
	FrameOverlap          int
	MaxFilterWidth        int
	NumFilterCoefficients int
}

func (c FingerprintConfig) ItemDuration() time.Duration {
	duration := c.FrameSize - c.FrameOverlap
	return time.Microsecond * time.Duration(duration * 1000000 / c.SampleRate)
}

func (c FingerprintConfig) Delay() time.Duration {
	delay := (c.FrameSize - c.FrameOverlap) * ((c.NumFilterCoefficients - 1) + (c.MaxFilterWidth- 1)) + c.FrameOverlap
	return time.Microsecond * time.Duration(delay * 1000000 / c.SampleRate)
}

func (c FingerprintConfig) Offset(i int) time.Duration {
	return c.ItemDuration() * time.Duration(i)
}

func (c FingerprintConfig) Duration(i int) time.Duration {
	if i == 0 {
		return time.Duration(0)
	}
	return c.Offset(i) + c.Delay()
}

var fingerprintConfigs = map[int]FingerprintConfig{
	1: {
		SampleRate:            11025,
		FrameSize:             4096,
		FrameOverlap:          4096 - 4096 / 3,
		NumFilterCoefficients: 5,
		MaxFilterWidth:        16,
	},
}

type MatchResult struct {
	Version      int
	Config       FingerprintConfig
	MasterLength int
	QueryLength  int
	Sections     []MatchingSection
}

func (mr MatchResult) MatchingDuration() time.Duration {
	length := 0
	for _, s := range mr.Sections {
		length += s.End - s.Start
	}
	return mr.Config.Duration(length)
}

func (mr MatchResult) QueryOffset() time.Duration {
	if len(mr.Sections) == 0 {
		return time.Duration(0)
	}
	s := mr.Sections[0]
	if s.Offset < 0 {
		return mr.Config.Offset(s.Start - s.Offset)
	} else {
		return mr.Config.Offset(s.Start)
	}
}

func (mr MatchResult) QueryDuration() time.Duration {
	return mr.Config.Duration(mr.QueryLength)
}

func (mr MatchResult) MasterOffset() time.Duration {
	if len(mr.Sections) == 0 {
		return time.Duration(0)
	}
	s := mr.Sections[0]
	if s.Offset > 0 {
		return mr.Config.Offset(s.Start + s.Offset)
	} else {
		return mr.Config.Offset(s.Start)
	}
}

func (mr MatchResult) MasterDuration() time.Duration {
	return mr.Config.Duration(mr.MasterLength)
}

type MatchingSection struct {
	Offset int
	Start  int
	End    int
	Score  float64
}

var ErrInvalidFingerprintVersion = errors.New("inv alid fingerprint version")

func MatchFingerprints(master *chromaprint.Fingerprint, query *chromaprint.Fingerprint) (*MatchResult, error) {
	if master.Version != query.Version {
		return nil, ErrInvalidFingerprintVersion
	}
	config, exists := fingerprintConfigs[master.Version]
	if !exists {
		return nil, ErrInvalidFingerprintVersion
	}

	if len(master.Hashes) >= 1<<16 {
		return nil, errors.New("master fingerprint too long")
	}
	if len(query.Hashes) >= 1<<16 {
		return nil, errors.New("query fingerprint too long")
	}

	result := &MatchResult{
		Version:      master.Version,
		Config:       config,
		MasterLength: len(master.Hashes),
		QueryLength:  len(query.Hashes),
	}

	offsetPeaks := alignFingerprints(master, query, NumOffsetCandidates)
	for _, peak := range offsetPeaks {
		sections, err := matchAlignedFingerprints(master, query, peak.Offset)
		if err != nil {
			return nil, errors.WithMessage(err, "matching failed")
		}
		if len(sections) > 0 {
			result.Sections = sections
			break
		}
	}

	return result, nil
}

func matchAlignedFingerprints(master *chromaprint.Fingerprint, query *chromaprint.Fingerprint, offset int) ([]MatchingSection, error) {
	masterHashes := master.Hashes
	queryHashes := query.Hashes
	if offset >= 0 {
		masterHashes = masterHashes[offset:]
	} else {
		queryHashes = queryHashes[-offset:]
	}

	n := len(masterHashes)
	if n > len(queryHashes) {
		n = len(queryHashes)
	}

	diff := make([]float64, n)
	for i := 0; i < n; i++ {
		diff[i] = float64(util.PopCount32(masterHashes[i] ^ queryHashes[i]))
	}
//	log.Print(diff)

	smoothedDiff := make([]float64, n)
	gaussianFilter(diff, smoothedDiff, 3.6, 5)
//	log.Print(smoothedDiff)

	smoothedDiffGradient := make([]float64, n)
	gradient(smoothedDiff, smoothedDiffGradient, 7)
	//log.Print(smoothedDiffGradient)

	edges := []int{0}
	for i := 1; i < n-1; i++ {
		x0 := math.Abs(smoothedDiffGradient[i-1])
		x1 := math.Abs(smoothedDiffGradient[i])
		x2 := math.Abs(smoothedDiffGradient[i+1])
		if x0 <= x1 && x2 < x1 {
			g := x1 / (1 + smoothedDiff[i]/4)
			if g > 3.0 {
	//			log.Printf("peak %v %v %v", i, x1, g)
				edges = append(edges, i)
			}
		}
	}
	edges = append(edges, n)

	matches := make([]MatchingSection, 0, len(edges)-1)
	for i := 0; i < len(edges)-1; i++ {
		m := MatchingSection{offset, edges[i], edges[i+1], 0}
		for j := m.Start; j < m.End; j++ {
			m.Score += diff[j]
		}
		m.Score /= float64(m.End - m.Start)
		if m.Score < 13 {
			matches = append(matches, m)
		}
	}

	return matches, nil
}

type OffsetHit struct {
	Offset int
	Count  int
}

func alignFingerprints(master *chromaprint.Fingerprint, query *chromaprint.Fingerprint, maxOffsets int) []OffsetHit {
	mask := hashBitMask(NumAlignBits)

	type HashOffset struct {
		Hash   uint32
		Offset int
	}

	queryHashes := make([]HashOffset, 0, len(query.Hashes))
	for offset, hash := range query.Hashes {
		queryHashes = append(queryHashes, HashOffset{hash & mask, offset})
	}
	sort.Slice(queryHashes, func(i, j int) bool { return queryHashes[i].Hash < queryHashes[j].Hash })

	masterHashes := make([]HashOffset, 0, len(master.Hashes))
	for offset, hash := range master.Hashes {
		masterHashes = append(masterHashes, HashOffset{hash & mask, offset})
	}
	sort.Slice(masterHashes, func(i, j int) bool { return masterHashes[i].Hash < masterHashes[j].Hash })

	offsets := make(map[int]int)
	maxOffsetCount := 0
	i := 0
	for _, mo := range masterHashes {
		for i < len(queryHashes) && queryHashes[i].Hash < mo.Hash {
			i++
		}
		if i >= len(queryHashes) {
			break
		}
		for j := i; j < len(queryHashes) && queryHashes[j].Hash == mo.Hash; j++ {
			offset := mo.Offset - queryHashes[j].Offset
			offsets[offset]++
			if offsets[offset] > maxOffsetCount {
				maxOffsetCount = offsets[offset]
			}
		}
	}

	// TODO gaussian filter

	offsetHits := make([]OffsetHit, 0)
	for offset, count := range offsets {
		if count >= maxOffsetCount/MaxOffsetThresholdDiv {
			if offsets[offset-1] <= count && offsets[offset+1] < count {
				offsetHits = append(offsetHits, OffsetHit{offset, count})
			}
		}
	}
	sort.Slice(offsetHits, func(i, j int) bool { return offsetHits[i].Count >= offsetHits[j].Count })
	if len(offsetHits) > maxOffsets {
		offsetHits = offsetHits[:maxOffsets]
	}
	//log.Println("offsetHits", offsetHits)

	return offsetHits
}

func boxFilter(src, dst []float64, w int) {
	if len(src) != len(dst) {
		panic("src and dst must have the same size")
	}

	if w == 0 || len(src) == 0 {
		return
	}

	n := len(src)

	wl := w / 2
	wr := w - wl

	// TODO optimize!!!

	for i := 0; i < n; i++ {
		var sum float64
		for j := i - wl; j < i+wr; j++ {
			k := j
			for k < 0 || k >= n {
				if k < 0 {
					k = -k
				} else {
					k = 2*n - k - 2
				}
			}
			sum += src[k]
		}
		dst[i] = sum / float64(w)
	}
}

func gaussianFilter(src, dst []float64, sigma float64, n int) {
	w := int(math.Floor(math.Sqrt(12*sigma*sigma/float64(n) + 1)))
	wl := w
	if w%2 == 0 {
		wl -= 1
	}
	wu := wl + 2

	m := int(math.Floor(0.5 + (12*sigma*sigma-float64(n*wl*wl+4*n*wl+3*n))/float64(-4*wl-4)))

	tmp := make([]float64, len(dst))
	copy(tmp, src)

	ptr1 := tmp[:]
	ptr2 := dst[:]

	for i := 0; i < n; i++ {
		if i < m {
			boxFilter(ptr1, ptr2, wl)
		} else {
			boxFilter(ptr1, ptr2, wu)
		}
		ptr3 := ptr1
		ptr2 = ptr1
		ptr1 = ptr3
		for j := 0; j < len(ptr2); j++ {
			ptr2[j] = 0
		}
	}

	if n%1 == 0 {
		copy(ptr1, ptr2)
	}
}

func gradient(src, dst []float64, h int) {
	if len(src) != len(dst) {
		panic("src and dst must have the same size")
	}

	n := len(src)

	if n <= h {
		for i := 0; i < n; i++ {
			dst[i] = src[n-1] - src[0]
		}
		return
	}

	hl := h / 2

	for i := 0; i < n; i++ {
		var i0 int
		if i < hl {
			i0 = 0
		} else if i+h >= n {
			i0 = n - 1 - h
		} else {
			i0 = i - hl
		}
		i1 := i0 + h
		dst[i] = src[i1] - src[i0]
	}
}
