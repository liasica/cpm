package i18n

import "os"

var lang string

func init() {
	lang = os.Getenv("CPM_LANG")
}

// T 根据当前语言返回英文或中文字符串
// 默认返回英文，设置 CPM_LANG=zh 时返回中文
func T(en, zh string) string {
	if lang == "zh" {
		return zh
	}
	return en
}
