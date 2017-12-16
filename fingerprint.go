package priv

import (
	"github.com/acoustid/go-acoustid/chromaprint"
)

const NumQueryBits = 26

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
