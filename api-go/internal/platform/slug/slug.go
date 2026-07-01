package slug

import (
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

var (
	reNonAlnum = regexp.MustCompile(`[^a-z0-9]+`)
	reDash     = regexp.MustCompile(`-{2,}`)
)

// cyrillic-to-latin transliteration table (Russian)
var cyrMap = map[rune]string{
	'а': "a", 'б': "b", 'в': "v", 'г': "g", 'д': "d",
	'е': "e", 'ё': "yo", 'ж': "zh", 'з': "z", 'и': "i",
	'й': "y", 'к': "k", 'л': "l", 'м': "m", 'н': "n",
	'о': "o", 'п': "p", 'р': "r", 'с': "s", 'т': "t",
	'у': "u", 'ф': "f", 'х': "kh", 'ц': "ts", 'ч': "ch",
	'ш': "sh", 'щ': "sch", 'ъ': "", 'ы': "y", 'ь': "",
	'э': "e", 'ю': "yu", 'я': "ya",
}

// MaxBaseLen is the max length of the base slug (leaves room for "-<uuid>" suffix = 36+1 chars).
const MaxBaseLen = 160

// Generate builds a URL-safe slug from the given parts joined by "-".
// The result is capped at MaxBaseLen characters.
func Generate(parts ...string) string {
	s := strings.Join(parts, "-")
	s = strings.ToLower(s)

	// transliterate cyrillic
	var b strings.Builder
	for _, r := range s {
		if lat, ok := cyrMap[r]; ok {
			b.WriteString(lat)
		} else {
			b.WriteRune(r)
		}
	}
	s = b.String()

	// decompose Unicode and strip combining marks
	s = norm.NFD.String(s)
	var clean strings.Builder
	for _, r := range s {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		clean.WriteRune(r)
	}
	s = clean.String()

	s = reNonAlnum.ReplaceAllString(s, "-")
	s = reDash.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")

	if len(s) > MaxBaseLen {
		s = s[:MaxBaseLen]
		s = strings.TrimRight(s, "-")
	}
	return s
}
