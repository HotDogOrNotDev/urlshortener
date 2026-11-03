package main

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/url"
	"sync"
)

const maxIDAttempts = 8

var (
	// ErrInvalidURL возвращается, если строка не является абсолютным http/https URL.
	ErrInvalidURL = errors.New("невалидный URL")
	// ErrNotFound возвращается, если короткий идентификатор не сохранён.
	ErrNotFound = errors.New("короткий URL не найден")
	// ErrGenerateID возвращается, если не удалось получить уникальный идентификатор.
	ErrGenerateID = errors.New("не удалось сгенерировать уникальный идентификатор")
)

// generateID подменяется в тестах, чтобы проверить повтор при коллизии.
var generateID = generateShortID

// URLShortener хранит соответствие коротких идентификаторов и исходных URL.
type URLShortener struct {
	urls map[string]string
	mu   sync.RWMutex
}

// NewURLShortener создаёт пустое хранилище в памяти.
func NewURLShortener() *URLShortener {
	return &URLShortener{
		urls: make(map[string]string),
	}
}

// Shorten проверяет URL, создаёт уникальный идентификатор длиной 6–8 символов и сохраняет его.
func (us *URLShortener) Shorten(originalURL string) (string, error) {
	if !isValidURL(originalURL) {
		return "", ErrInvalidURL
	}

	us.mu.Lock()
	defer us.mu.Unlock()

	for i := 0; i < maxIDAttempts; i++ {
		shortID := generateID()
		if _, exists := us.urls[shortID]; exists {
			continue
		}
		us.urls[shortID] = originalURL
		return shortID, nil
	}
	return "", ErrGenerateID
}

// GetOriginal возвращает исходный URL по короткому идентификатору.
func (us *URLShortener) GetOriginal(shortID string) (string, error) {
	us.mu.RLock()
	defer us.mu.RUnlock()

	original, ok := us.urls[shortID]
	if !ok {
		return "", ErrNotFound
	}
	return original, nil
}

// generateShortID возвращает случайную URL-safe строку длиной 8 символов.
func generateShortID() string {
	buf := make([]byte, 6)
	_, _ = rand.Read(buf)
	return base64.RawURLEncoding.EncodeToString(buf)
}

// isValidURL принимает только абсолютные адреса со схемой http или https и непустым хостом.
func isValidURL(str string) bool {
	u, err := url.Parse(str)
	if err != nil || u == nil {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	return u.Host != ""
}
