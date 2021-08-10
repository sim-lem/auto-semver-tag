package semver

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSemverParsing(t *testing.T) {
	var tests = []struct {
		name          string
		strToParse    string
		expectedRes   SemVer
		expectedError error
	}{
		{
			name:          "Valid version with 'v': v1.0.0",
			strToParse:    "v1.0.0",
			expectedRes:   SemVer{1, 0, 0},
			expectedError: nil,
		},
		{
			name:          "Valid version without 'v': 1.0.0",
			strToParse:    "1.0.0",
			expectedRes:   SemVer{1, 0, 0},
			expectedError: nil,
		},
		{
			name:          "Not valid version with leading '0': 01.0.0",
			strToParse:    "01.0.0",
			expectedRes:   SemVer{0, 0, 0},
			expectedError: fmt.Errorf("invalid semver: %s", "01.0.0"),
		},
		{
			name:          "Valid version with prerelease: 1.0.1-rc.1",
			strToParse:    "v1.0.1-rc.1",
			expectedRes:   SemVer{1, 0, 1},
			expectedError: nil,
		},
		{
			name:          "Valid version with build: 1.0.1+build.1",
			strToParse:    "1.0.1+build.1",
			expectedRes:   SemVer{1, 0, 1},
			expectedError: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			res, err := New(test.strToParse)
			if err != nil {
				assert.Equal(t, test.expectedError, err)
				return
			}

			if test.expectedError != nil {
				t.Fatalf("No error returned. But expected: %v", test.expectedError)
			}

			assert.Equal(t, test.expectedRes.String(), res.String())

		})
	}
}

func TestSemverIncrement(t *testing.T) {
	var tests = []struct {
		name          string
		semver        SemVer
		expectedRes   SemVer
		incrementType string
	}{
		{
			name:          "Increment major",
			semver:        SemVer{1, 0, 0},
			expectedRes:   SemVer{2, 0, 0},
			incrementType: IncrementTypeMajor,
		},
		{
			name:          "Increment minor",
			semver:        SemVer{1, 0, 0},
			expectedRes:   SemVer{1, 1, 0},
			incrementType: IncrementTypeMinor,
		},
		{
			name:          "Increment patch",
			semver:        SemVer{1, 0, 0},
			expectedRes:   SemVer{1, 0, 1},
			incrementType: IncrementTypePatch,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			res := test.semver.IncrementVersion(test.incrementType)
			assert.Equal(t, test.expectedRes.String(), res.String())
		})
	}
}
