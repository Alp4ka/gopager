package gopager

const (
	NoLimit      = -1
	MaxLimit     = 100
	DefaultLimit = 10
)

func IsNormalizedLimitMax(limit int, maxLimit int) (int, bool) {
	if limit <= 0 {
		return DefaultLimit, false
	} else if limit > maxLimit {
		return maxLimit, false
	}

	return limit, true
}

func NormalizeLimitMax(limit int, maxLimit int) int {
	ret, _ := IsNormalizedLimitMax(limit, maxLimit)
	return ret
}

func NormalizeLimit(limit int) int {
	return NormalizeLimitMax(limit, MaxLimit)
}
