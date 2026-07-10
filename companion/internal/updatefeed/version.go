package updatefeed

import (
	"fmt"
	"strconv"
	"strings"
)

type Version struct {
	major      uint64
	minor      uint64
	patch      uint64
	prerelease []string
}

func ParseVersion(value string) (Version, error) {
	value = strings.TrimSpace(value)
	withoutBuild, _, _ := strings.Cut(value, "+")
	core, prerelease, hasPrerelease := strings.Cut(withoutBuild, "-")
	parts := strings.Split(core, ".")
	if len(parts) != 3 {
		return Version{}, fmt.Errorf("%q is not a semantic version", value)
	}
	numbers := make([]uint64, 3)
	for index, part := range parts {
		if part == "" || (len(part) > 1 && part[0] == '0') {
			return Version{}, fmt.Errorf("%q is not a semantic version", value)
		}
		parsed, err := strconv.ParseUint(part, 10, 64)
		if err != nil {
			return Version{}, fmt.Errorf("%q is not a semantic version", value)
		}
		numbers[index] = parsed
	}
	version := Version{major: numbers[0], minor: numbers[1], patch: numbers[2]}
	if !hasPrerelease {
		return version, nil
	}
	if prerelease == "" {
		return Version{}, fmt.Errorf("%q is not a semantic version", value)
	}
	for _, identifier := range strings.Split(prerelease, ".") {
		if identifier == "" || !validIdentifier(identifier) {
			return Version{}, fmt.Errorf("%q is not a semantic version", value)
		}
		if numeric(identifier) && len(identifier) > 1 && identifier[0] == '0' {
			return Version{}, fmt.Errorf("%q is not a semantic version", value)
		}
		version.prerelease = append(version.prerelease, identifier)
	}
	return version, nil
}

func (version Version) Compare(other Version) int {
	for _, pair := range [][2]uint64{{version.major, other.major}, {version.minor, other.minor}, {version.patch, other.patch}} {
		if pair[0] < pair[1] {
			return -1
		}
		if pair[0] > pair[1] {
			return 1
		}
	}
	if len(version.prerelease) == 0 && len(other.prerelease) == 0 {
		return 0
	}
	if len(version.prerelease) == 0 {
		return 1
	}
	if len(other.prerelease) == 0 {
		return -1
	}
	limit := min(len(version.prerelease), len(other.prerelease))
	for index := 0; index < limit; index++ {
		left, right := version.prerelease[index], other.prerelease[index]
		if left == right {
			continue
		}
		leftNumeric, rightNumeric := numeric(left), numeric(right)
		switch {
		case leftNumeric && rightNumeric:
			leftValue, _ := strconv.ParseUint(left, 10, 64)
			rightValue, _ := strconv.ParseUint(right, 10, 64)
			if leftValue < rightValue {
				return -1
			}
			return 1
		case leftNumeric:
			return -1
		case rightNumeric:
			return 1
		case left < right:
			return -1
		default:
			return 1
		}
	}
	if len(version.prerelease) < len(other.prerelease) {
		return -1
	}
	if len(version.prerelease) > len(other.prerelease) {
		return 1
	}
	return 0
}

func validIdentifier(value string) bool {
	for _, character := range value {
		if (character >= 'a' && character <= 'z') ||
			(character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') || character == '-' {
			continue
		}
		return false
	}
	return true
}

func numeric(value string) bool {
	for _, character := range value {
		if character < '0' || character > '9' {
			return false
		}
	}
	return value != ""
}
