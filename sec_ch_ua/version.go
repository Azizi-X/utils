package sec_ch_ua

import (
	"strconv"
	"strings"
)

type Version struct {
	components []uint32
}

func (v Version) IsValid() bool {
	return len(v.components) > 0
}

func (v Version) Components() []uint32 {
	return v.components
}

func NewVersion(versionStr string) Version {
	var parsed []uint32

	if !parseVersionNumbers(versionStr, &parsed) {
		return Version{}
	}

	return Version{
		components: parsed,
	}
}

func parseVersionNumbers(versionStr string, parsed *[]uint32) bool {
	if versionStr == "" {
		return false
	}

	numbers := strings.Split(versionStr, ".")

	if len(numbers) == 0 {
		return false
	}

	for i, numStr := range numbers {
		if strings.HasPrefix(numStr, "+") {
			return false
		}

		num64, err := strconv.ParseUint(numStr, 10, 32)
		if err != nil {
			return false
		}

		num := uint32(num64)
		if i == 0 {
			if strconv.FormatUint(uint64(num), 10) != numStr {
				return false
			}
		}

		*parsed = append(*parsed, num)
	}

	return true
}
