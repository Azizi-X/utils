package text

import (
	"strings"
	"unicode/utf8"

	"github.com/eskriett/confusables"
	"golang.org/x/text/unicode/norm"
)

func StripDiacritics(value string) string {
	value = norm.NFD.String(value)
	value = strings.Map(func(r rune) rune {
		if r >= '\u0300' && r <= '\u036f' {
			return -1
		}
		return r
	}, value)
	return norm.NFC.String(value)
}

func Normalize(value string) string {
	return strings.ToLower(norm.NFD.String(confusables.ToSkeleton(value)))
}

func FuzzyMatch(needle, value string) bool {
	for _, char := range needle {
		index := strings.IndexRune(value, char)
		if index < 0 {
			return false
		}
		value = value[index+utf8.RuneLen(char):]
	}
	return true
}

func StartsWith(query string, values ...string) bool {
	query = strings.ToLower(query)
	for _, value := range values {
		value = strings.ToLower(value)
		if strings.HasPrefix(value, query) || strings.HasPrefix(StripDiacritics(value), query) {
			return true
		}
	}
	return false
}

func StartsWithNormalized(query string, values ...string) bool {
	query = Normalize(strings.ToLower(query))
	for _, value := range values {
		if strings.HasPrefix(Normalize(StripDiacritics(strings.ToLower(value))), query) {
			return true
		}
	}
	return false
}

func FuzzyMatchesAny(query string, values ...string) bool {
	query = strings.ToLower(query)
	normalizedQuery := Normalize(query)
	for _, value := range values {
		value = StripDiacritics(strings.ToLower(value))
		if FuzzyMatch(query, value) || FuzzyMatch(normalizedQuery, Normalize(value)) {
			return true
		}
	}
	return false
}
