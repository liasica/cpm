package i18n

import "os"

var lang string

func init() {
	lang = os.Getenv("CPM_LANG")
}

// T returns the English or Chinese string based on the CPM_LANG env var.
// Defaults to English; set CPM_LANG=zh for Chinese.
func T(en, zh string) string {
	if lang == "zh" {
		return zh
	}
	return en
}
