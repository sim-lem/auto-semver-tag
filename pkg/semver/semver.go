package semver

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// IncrementType allow alphanumeric comparison: major < minor < patch < unknown
type IncrementType string

const (
	// https://ihateregex.io/expr/semver
	SemVerRegExp = `^v?(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)` +
		`(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)` +
		`(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?` +
		`(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`

	IncrementTypeMajor   IncrementType = "major"
	IncrementTypeMinor   IncrementType = "minor"
	IncrementTypePatch   IncrementType = "patch"
	IncrementTypeUnknown IncrementType = "unknown"
)

func StringToIncrementType(val string) IncrementType {
	switch val {
	case string(IncrementTypeMajor):
		return IncrementTypeMajor
	case string(IncrementTypeMinor):
		return IncrementTypeMinor
	case string(IncrementTypePatch):
		return IncrementTypePatch
	default:
		return IncrementTypeUnknown
	}
}

type SemVer struct {
	major uint64
	minor uint64
	patch uint64
}

func newInvalidSemVerError(semVer string) (SemVer, error) {
	return SemVer{}, fmt.Errorf("invalid semver: %s", semVer)
}

func New(semVer string) (SemVer, error) {

	isSemVerValid, err := regexp.MatchString(SemVerRegExp, semVer)
	if err != nil || !isSemVerValid {
		return newInvalidSemVerError(semVer)
	}

	if strings.Index(semVer, "v") == 0 {
		semVer = semVer[1:]
	}

	parts := strings.SplitN(semVer, ".", 3)

	major, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return newInvalidSemVerError(semVer)
	}

	minor, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil {
		return newInvalidSemVerError(semVer)
	}

	patchStr := parts[2]

	if buildIndex := strings.IndexRune(patchStr, '+'); buildIndex != -1 {
		patchStr = patchStr[:buildIndex]
	}

	if preIndex := strings.IndexRune(patchStr, '-'); preIndex != -1 {
		patchStr = patchStr[:preIndex]
	}

	patch, err := strconv.ParseUint(patchStr, 10, 64)
	if err != nil {
		return newInvalidSemVerError(semVer)
	}

	res := SemVer{
		major,
		minor,
		patch,
	}

	return res, err
}

func (s SemVer) IncrementVersion(incrementType IncrementType) SemVer {
	switch incrementType {
	case IncrementTypeMajor:
		return SemVer{
			major: s.major + 1,
			minor: 0,
			patch: 0,
		}

	case IncrementTypeMinor:
		return SemVer{
			major: s.major,
			minor: s.minor + 1,
			patch: 0,
		}
	case IncrementTypePatch:
		return SemVer{
			major: s.major,
			minor: s.minor,
			patch: s.patch + 1,
		}
	default:
		panic("invalid increment type")
	}
}

func (s SemVer) String() string {
	return fmt.Sprintf("v%d.%d.%d", s.major, s.minor, s.patch)
}

func (s SemVer) IsGreaterThan(o SemVer) bool {
	if s.major > o.major {
		return true
	}
	if s.major < o.major {
		return false
	}

	// Major versions are equal

	if s.minor > o.minor {
		return true
	}
	if s.minor < o.minor {
		return false
	}

	// Major and minor versions are equal

	if s.patch > o.patch {
		return true
	}
	return false
}
