package gemini_api

import (
	"log"
	"os"
	"strings"
	"sync"
)

var (
	apiKeys    []string
	currentIdx int
	mu         sync.Mutex
)

func LoadAPIKeys() {
	keys := os.Getenv("GEMINI_API_KEYS")
	if keys == "" {
		log.Fatal("❌ Không tìm thấy GEMINI_API_KEYS trong biến môi trường")
	}

	apiKeys = splitKeys(keys)
	currentIdx = 0
	log.Printf("🔑 Đã nạp %d API keys vào pool", len(apiKeys))
}

func splitKeys(keys string) []string {
	var keyList []string
	for _, key := range strings.Split(keys, ",") {
		keyList = append(keyList, strings.TrimSpace(key))
	}
	return keyList
}

func GetNextAPIKey() string {
	mu.Lock()
	defer mu.Unlock()

	if len(apiKeys) == 0 {
		log.Fatal("❌ Không có API keys khả dụng")
	}

	key := apiKeys[currentIdx]
	currentIdx = (currentIdx + 1) % len(apiKeys)
	return key
}

func RemoveAPIKey(failedKey string) {
	mu.Lock()
	defer mu.Unlock()

	for i, key := range apiKeys {
		if key == failedKey {
			log.Printf("⚠ API key lỗi: %s -> Xóa khỏi danh sách", failedKey)
			apiKeys = append(apiKeys[:i], apiKeys[i+1:]...)
			break
		}
	}
}
