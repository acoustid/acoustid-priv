package priv

import "github.com/acoustid/go-acoustid/chromaprint"

func ExtractQuery(fp *chromaprint.Fingerprint, nbits int) []int32 {
	var mask uint32 = 0xaaaaaaaa
	if nbits <= 16 {
		mask &= (1 << (uint(nbits) * 2)) - 1
	} else {
		mask |= 0x55555555 & ((1 << (uint(nbits - 16) * 2)) - 1)
	}
	query := make([]int32, len(fp.Hashes))
	for i := 0; i < len(fp.Hashes); i++ {
		query[i] = int32(fp.Hashes[i] & mask)
	}
	return query
}
