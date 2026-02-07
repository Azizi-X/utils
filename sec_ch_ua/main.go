package sec_ch_ua

import (
	"fmt"
	"strconv"
	"strings"
)

var (
	KFullVersion  UserAgentBrandVersionType = 1
	KMajorVersion UserAgentBrandVersionType = 2
)

type UserAgentBrandVersion struct {
	Brand   string
	Version string
}

type UserAgentBrandVersionType int

func SerializeBrandVersionList(list []UserAgentBrandVersion) string {
	var parts []string

	for _, bv := range list {
		brand := serializeSHString(bv.Brand)
		if bv.Version == "" {
			parts = append(parts, fmt.Sprintf(`"%s"`, brand))
		} else {
			version := serializeSHString(bv.Version)
			parts = append(parts, fmt.Sprintf(`"%s";v="%s"`, brand, version))
		}
	}

	return strings.Join(parts, ", ")
}

func getGreasedUserAgentBrandVersion(
	seed int,
	outputVersionType UserAgentBrandVersionType,
) UserAgentBrandVersion {

	greaseyChars := []string{
		" ", "(", ":", "-", ".", "/",
		")", ";", "=", "?", "_",
	}

	greasedVersions := []string{
		"8", "99", "24",
	}

	greaseyBrand := "Not" +
		greaseyChars[seed%len(greaseyChars)] +
		"A" +
		greaseyChars[(seed+1)%len(greaseyChars)] +
		"Brand"

	greaseyVersion := greasedVersions[seed%len(greasedVersions)]

	return getProcessedGreasedBrandVersion(
		greaseyBrand,
		greaseyVersion,
		outputVersionType,
	)
}

func getProcessedGreasedBrandVersion(
	greaseyBrand string,
	greaseyVersion string,
	outputVersionType UserAgentBrandVersionType,
) UserAgentBrandVersion {
	version := NewVersion(greaseyVersion)
	if !version.IsValid() {
		panic("invalid version")
	}

	components := version.Components()

	var greaseyMajorVersion string
	var greaseyFullVersion string

	if len(components) > 1 {
		greaseyMajorVersion = strconv.FormatUint(uint64(components[0]), 10)
		greaseyFullVersion = greaseyVersion
	} else {
		greaseyMajorVersion = greaseyVersion
		greaseyFullVersion = greaseyVersion + ".0.0.0"
	}

	outputVersion := greaseyMajorVersion
	if outputVersionType == KFullVersion {
		outputVersion = greaseyFullVersion
	}

	return UserAgentBrandVersion{
		Brand:   greaseyBrand,
		Version: outputVersion,
	}
}

func getRandomOrder(seed int, size int) []int {
	if size < 2 {
		panic("CHECK_GE failed: size < 2")
	} else if size > 4 {
		panic("CHECK_LE failed: size > 4")
	}

	switch size {
	case 2:
		return []int{
			seed % size,
			(seed + 1) % size,
		}
	case 3:
		orders := [6][3]int{
			{0, 1, 2},
			{0, 2, 1},
			{1, 0, 2},
			{1, 2, 0},
			{2, 0, 1},
			{2, 1, 0},
		}

		order := orders[seed%len(orders)]

		return []int{
			order[0],
			order[1],
			order[2],
		}

	default:
		orders := [24][4]int{
			{0, 1, 2, 3}, {0, 1, 3, 2}, {0, 2, 1, 3}, {0, 2, 3, 1},
			{0, 3, 1, 2}, {0, 3, 2, 1}, {1, 0, 2, 3}, {1, 0, 3, 2},
			{1, 2, 0, 3}, {1, 2, 3, 0}, {1, 3, 0, 2}, {1, 3, 2, 0},
			{2, 0, 1, 3}, {2, 0, 3, 1}, {2, 1, 0, 3}, {2, 1, 3, 0},
			{2, 3, 0, 1}, {2, 3, 1, 0}, {3, 0, 1, 2}, {3, 0, 2, 1},
			{3, 1, 0, 2}, {3, 1, 2, 0}, {3, 2, 0, 1}, {3, 2, 1, 0},
		}

		order := orders[seed%len(orders)]

		return []int{
			order[0],
			order[1],
			order[2],
			order[3],
		}
	}
}

func shuffleBrandList(
	brandVersionList []UserAgentBrandVersion,
	seed int,
) []UserAgentBrandVersion {
	order := getRandomOrder(seed, len(brandVersionList))

	if len(brandVersionList) != len(order) {
		panic("CHECK_EQ failed: size mismatch")
	}

	shuffled := make([]UserAgentBrandVersion, len(brandVersionList))

	for i := range order {
		shuffled[order[i]] = brandVersionList[i]
	}

	return shuffled
}

func GenerateBrandVersionList(
	seed int,
	brand string,
	version string,
	outputVersionType UserAgentBrandVersionType,
	additionalBrandVersion *UserAgentBrandVersion,
) []UserAgentBrandVersion {
	if seed < 0 {
		panic("seed must be >= 0")
	}

	greaseyBV := getGreasedUserAgentBrandVersion(seed, outputVersionType)

	chromiumBV := UserAgentBrandVersion{
		Brand:   "Chromium",
		Version: version,
	}

	brandVersionList := []UserAgentBrandVersion{
		greaseyBV,
		chromiumBV,
	}

	if brand != "" {
		brandVersionList = append(
			brandVersionList,
			UserAgentBrandVersion{
				Brand:   brand,
				Version: version,
			},
		)
	}

	if additionalBrandVersion != nil {
		brandVersionList = append(
			brandVersionList,
			*additionalBrandVersion,
		)
	}

	return shuffleBrandList(brandVersionList, seed)
}
