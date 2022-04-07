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
		{
			name:          "Valid version with multi-digits: v12.34.56",
			strToParse:    "v12.34.56",
			expectedRes:   SemVer{12, 34, 56},
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
	v567 := SemVer{5, 6, 7}

	tests := []struct {
		before    SemVer
		after     SemVer
		increment IncrementType
	}{
		{
			before:    SemVer{},
			after:     SemVer{1, 0, 0},
			increment: IncrementTypeMajor,
		},
		{
			before:    SemVer{},
			after:     SemVer{0, 1, 0},
			increment: IncrementTypeMinor,
		},
		{
			before:    SemVer{},
			after:     SemVer{0, 0, 1},
			increment: IncrementTypePatch,
		},

		{
			before:    v567,
			after:     SemVer{6, 0, 0},
			increment: IncrementTypeMajor,
		},
		{
			before:    v567,
			after:     SemVer{5, 7, 0},
			increment: IncrementTypeMinor,
		},
		{
			before:    v567,
			after:     SemVer{5, 6, 8},
			increment: IncrementTypePatch,
		},
	}

	for _, tt := range tests {
		name := fmt.Sprintf("Increment %s from %s to %s", tt.increment, tt.before, tt.after)
		t.Run(name, func(t *testing.T) {
			res := tt.before.IncrementVersion(tt.increment)
			assert.Equal(t, tt.after.String(), res.String())
		})
	}
}

func TestStringToIncrementType(t *testing.T) {
	tests := []struct {
		val  string
		want IncrementType
	}{
		{
			val:  "major",
			want: IncrementTypeMajor,
		},
		{
			val:  "minor",
			want: IncrementTypeMinor,
		},
		{
			val:  "patch",
			want: IncrementTypePatch,
		},
		{
			val:  "Major",
			want: IncrementTypeUnknown,
		},
		{
			val:  "something else",
			want: IncrementTypeUnknown,
		},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s is %s", tt.val, tt.want), func(t *testing.T) {
			assert.Equal(t, tt.want, StringToIncrementType(tt.val))
		})
	}
}
