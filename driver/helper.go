package driver

import "strings"

// GetCaseInsensitiveMap coercs the map's keys to lower case, which only works
// when unicode char is in ASCII subset. May overwrite key-value pairs on
// different permutations of key case as in Key and key. DON'T force values to the
// lower case unconditionally, because values for keys such as mountpoint or
// keylocation are case-sensitive.
// Note that although keys such as 'comPREssion' are accepted and processed,
// even if they are technically invalid, updates to rectify such typing will be
// prohibited as a forbidden update.
func GetCaseInsensitiveMap(dict *map[string]string) map[string]string {
	insensitiveDict := map[string]string{}

	for k, v := range *dict {
		insensitiveDict[strings.ToLower(k)] = v
	}
	return insensitiveDict
}

// size constants
const (
	MB = 1000 * 1000
	GB = 1000 * 1000 * 1000
	Mi = 1024 * 1024
	Gi = 1024 * 1024 * 1024
)

// getRoundedCapacity rounds the capacity on 1024 base
func getRoundedCapacity(size int64) int64 {

	/*
	 * volblocksize and recordsize must be power of 2 from 512B to 1M
	 * so keeping the size in the form of Gi or Mi should be
	 * sufficient to make volsize multiple of volblocksize/recordsize.
	 */
	if size > Gi {
		return ((size + Gi - 1) / Gi) * Gi
	}

	// Keeping minimum allocatable size as 1Mi (1024 * 1024)
	return ((size + Mi - 1) / Mi) * Mi
}
