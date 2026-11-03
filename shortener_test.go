package main

import (
	"errors"
	"regexp"
	"testing"
)

func TestURLShortener_Shorten(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"валидный HTTP URL", "http://example.com", false},
		{"валидный HTTPS URL", "https://google.com/search?q=test", false},
		{"HTTPS с портом", "https://example.com:8443/path", false},
		{"схема в верхнем регистре", "HTTP://example.com/a", false},
		{"невалидный URL", "not-a-url", true},
		{"пустая строка", "", true},
		{"схема ftp", "ftp://example.com/file", true},
		{"без хоста", "http://", true},
		{"пробел в адресе", "http://exa mple.com", true},
	}

	shortener := NewURLShortener()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shortID, err := shortener.Shorten(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ошибка = %v, ожидали ошибку = %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				if !errors.Is(err, ErrInvalidURL) {
					t.Errorf("ошибка = %v, ожидали ErrInvalidURL", err)
				}
				return
			}
			if len(shortID) < 6 || len(shortID) > 8 {
				t.Errorf("длина короткого ID = %d, ожидалась 6–8: %s", len(shortID), shortID)
			}
			got, err := shortener.GetOriginal(shortID)
			if err != nil {
				t.Fatalf("GetOriginal() ошибка = %v", err)
			}
			if got != tt.url {
				t.Errorf("GetOriginal() = %q, ожидали %q", got, tt.url)
			}
		})
	}
}

func TestURLShortener_GetOriginal(t *testing.T) {
	shortener := NewURLShortener()
	const original = "https://example.com/page"
	id, err := shortener.Shorten(original)
	if err != nil {
		t.Fatalf("Shorten() ошибка = %v", err)
	}

	tests := []struct {
		name    string
		shortID string
		want    string
		wantErr error
	}{
		{"существующий ID", id, original, nil},
		{"отсутствующий ID", "missing", "", ErrNotFound},
		{"пустой ID", "", "", ErrNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := shortener.GetOriginal(tt.shortID)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ошибка = %v, ожидали %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("GetOriginal() = %q, ожидали %q", got, tt.want)
			}
		})
	}
}

func TestGenerateShortID(t *testing.T) {
	validID := regexp.MustCompile(`^[A-Za-z0-9_-]{6,8}$`)
	seen := make(map[string]struct{})

	for i := 0; i < 20; i++ {
		id := generateShortID()
		if !validID.MatchString(id) {
			t.Fatalf("идентификатор %q не соответствует формату 6–8 URL-safe символов", id)
		}
		seen[id] = struct{}{}
	}
	if len(seen) < 20 {
		t.Fatalf("ожидались уникальные идентификаторы, получили %d уникальных из 20", len(seen))
	}
}

func TestURLShortener_ShortenCollision(t *testing.T) {
	orig := generateID
	t.Cleanup(func() { generateID = orig })

	t.Run("повтор при коллизии", func(t *testing.T) {
		us := NewURLShortener()
		us.urls["collide"] = "https://already.example"

		calls := 0
		generateID = func() string {
			calls++
			if calls == 1 {
				return "collide"
			}
			return "unique1"
		}

		id, err := us.Shorten("https://example.com/new")
		if err != nil {
			t.Fatalf("Shorten() ошибка = %v", err)
		}
		if id != "unique1" {
			t.Fatalf("Shorten() = %q, ожидали unique1", id)
		}
		if us.urls["collide"] != "https://already.example" {
			t.Fatal("существующая запись была перезаписана")
		}
	})

	t.Run("исчерпаны попытки", func(t *testing.T) {
		us := NewURLShortener()
		us.urls["same"] = "https://already.example"
		generateID = func() string { return "same" }

		_, err := us.Shorten("https://example.com/new")
		if !errors.Is(err, ErrGenerateID) {
			t.Fatalf("ошибка = %v, ожидали ErrGenerateID", err)
		}
	})
}
