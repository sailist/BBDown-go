package bvconv

import (
	"errors"
	"strings"
)

const (
	xorCode  = 23442827791579
	maskCode = (1 << 51) - 1
	maxAid   = maskCode + 1
	minAid   = 1
	base     = 58
	bvLen    = 9
)

var (
	alphabet    = []byte("FcwAPNKTMug3GV5Lj7EJnHpWsx4tb8haYeviqBz6rkCy12mUSDQX9RdoZf")
	revAlphabet [256]int64
)

func init() {
	for i := 0; i < len(alphabet); i++ {
		revAlphabet[alphabet[i]] = int64(i)
	}
}

// Encode converts an AID (avid) to a BV string.
// It panics for invalid aid values, matching the original C# behavior.
func Encode(aid int64) string {
	if aid < minAid {
		panic("aid is too small")
	}
	if aid >= maxAid {
		panic("aid is too big")
	}

	bvid := make([]byte, bvLen)
	tmp := (maxAid | aid) ^ xorCode

	for i := bvLen - 1; tmp != 0; i-- {
		bvid[i] = alphabet[tmp%base]
		tmp /= base
	}

	bvid[0], bvid[6] = bvid[6], bvid[0]
	bvid[1], bvid[4] = bvid[4], bvid[1]

	return "BV1" + string(bvid)
}

// Decode converts a BV string to an AID (avid).
// It accepts both the full "BV1xxxxx" form and the 9-char body.
func Decode(bv string) (int64, error) {
	bv = strings.TrimSpace(bv)

	if strings.HasPrefix(bv, "BV1") || strings.HasPrefix(bv, "bv1") || strings.HasPrefix(bv, "Bv1") {
		bv = bv[3:]
	}

	if len(bv) != bvLen {
		return 0, errors.New("invalid bv length")
	}

	bvid := []byte(bv)
	bvid[0], bvid[6] = bvid[6], bvid[0]
	bvid[1], bvid[4] = bvid[4], bvid[1]

	var aid int64
	for _, b := range bvid {
		idx := revAlphabet[b]
		if idx == 0 && b != alphabet[0] {
			return 0, errors.New("invalid bv character")
		}
		aid = aid*base + idx
	}

	return (aid & maskCode) ^ xorCode, nil
}
