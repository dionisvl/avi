package email

const (
	LocaleRU = "ru"
	LocaleEN = "en"
)

func PickLocale(loc string) string {
	switch loc {
	case LocaleRU:
		return LocaleRU
	default:
		return LocaleEN
	}
}
