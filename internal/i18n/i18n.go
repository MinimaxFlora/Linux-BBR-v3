// Package i18n 提供中英文语言包与全局语言切换（环境变量 BBRV3_LANG / /etc/bbrv3/lang 持久化）。
package i18n

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

// Lang 语言标识。
type Lang string

const (
	Zh Lang = "zh"
	En Lang = "en"
)

var (
	mu      sync.RWMutex
	current = Zh
)

// zh / en 语言包。key 为 UI 文案标识。
var (
	zh = map[string]string{}
	en = map[string]string{}
)

// Set 切换语言。
func Set(l Lang) {
	mu.Lock()
	defer mu.Unlock()
	current = l
}

// Get 返回当前语言。
func Get() Lang {
	mu.RLock()
	defer mu.RUnlock()
	return current
}

// IsEn 当前是否为英文。
func IsEn() bool { return Get() == En }

// T 返回当前语言的文案。
func T(key string) string {
	mu.RLock()
	lang := current
	mu.RUnlock()
	m := zh
	if lang == En {
		m = en
	}
	if s, ok := m[key]; ok && s != "" {
		return s
	}
	// 回退到中文，再回退到 key 本身
	if s, ok := zh[key]; ok {
		return s
	}
	return key
}

// Tf 返回格式化后的文案。
func Tf(key string, args ...any) string {
	return fmt.Sprintf(T(key), args...)
}

// Init 初始化语言：环境变量 BBRV3_LANG 优先，其次 /etc/bbrv3/lang 文件，默认 zh。
func Init() {
	lang := ""
	if v := os.Getenv("BBRV3_LANG"); v != "" {
		lang = v
	} else if data, err := os.ReadFile("/etc/bbrv3/lang"); err == nil {
		lang = strings.TrimSpace(string(data))
	}
	switch strings.ToLower(lang) {
	case "en", "english":
		Set(En)
	default:
		Set(Zh)
	}
}

// Persist 把当前语言写入 /etc/bbrv3/lang（失败静默）。
func Persist() {
	l := Get()
	if err := os.MkdirAll("/etc/bbrv3", 0o755); err != nil {
		return
	}
	_ = os.WriteFile("/etc/bbrv3/lang", []byte(l+"\n"), 0o644)
}

// Register 注册语言包条目（由 app 包 init 调用）。
func Register(lang Lang, entries map[string]string) {
	mu.Lock()
	defer mu.Unlock()
	switch lang {
	case En:
		for k, v := range entries {
			en[k] = v
		}
	default:
		for k, v := range entries {
			zh[k] = v
		}
	}
}
