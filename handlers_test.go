package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestShortenHandler(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		body       string
		wantStatus int
		wantError  string
		wantURL    string
	}{
		{
			name:       "успешное сокращение",
			method:     http.MethodPost,
			body:       `{"url":"https://example.com/very/long/path"}`,
			wantStatus: http.StatusOK,
			wantURL:    "https://example.com/very/long/path",
		},
		{
			name:       "некорректный JSON",
			method:     http.MethodPost,
			body:       `{`,
			wantStatus: http.StatusBadRequest,
			wantError:  "некорректный JSON",
		},
		{
			name:       "невалидный URL",
			method:     http.MethodPost,
			body:       `{"url":"not-a-url"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "невалидный URL",
		},
		{
			name:       "пустой URL",
			method:     http.MethodPost,
			body:       `{"url":""}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "невалидный URL",
		},
		{
			name:       "метод не POST",
			method:     http.MethodGet,
			body:       `{"url":"https://example.com"}`,
			wantStatus: http.StatusMethodNotAllowed,
			wantError:  "метод не поддерживается",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := newMux(NewURLShortener())
			req := httptest.NewRequest(tt.method, "/shorten", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			mux.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Fatalf("статус = %d, ожидали %d, тело = %s", rr.Code, tt.wantStatus, rr.Body.String())
			}
			if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
				t.Fatalf("Content-Type = %q, ожидали application/json", ct)
			}

			if tt.wantError != "" {
				var payload errorResponse
				if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
					t.Fatalf("разбор JSON: %v", err)
				}
				if payload.Error != tt.wantError {
					t.Fatalf("error = %q, ожидали %q", payload.Error, tt.wantError)
				}
				return
			}

			var payload shortenResponse
			if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
				t.Fatalf("разбор JSON: %v", err)
			}
			if payload.OriginalURL != tt.wantURL {
				t.Fatalf("original_url = %q, ожидали %q", payload.OriginalURL, tt.wantURL)
			}
			if len(payload.ShortURL) < 6 || len(payload.ShortURL) > 8 {
				t.Fatalf("short_url = %q, ожидалась длина 6–8", payload.ShortURL)
			}
		})
	}
}

func TestRedirectHandler(t *testing.T) {
	const original = "https://example.com/target?q=1"

	us := NewURLShortener()
	shortID, err := us.Shorten(original)
	if err != nil {
		t.Fatalf("Shorten() ошибка = %v", err)
	}
	mux := newMux(us)

	t.Run("редирект на исходный URL", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/"+shortID, nil)
		rr := httptest.NewRecorder()

		mux.ServeHTTP(rr, req)

		if rr.Code != http.StatusFound {
			t.Fatalf("статус = %d, ожидали %d, тело = %s", rr.Code, http.StatusFound, rr.Body.String())
		}
		if loc := rr.Header().Get("Location"); loc != original {
			t.Fatalf("Location = %q, ожидали %q", loc, original)
		}
	})

	t.Run("неизвестный идентификатор", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/missing", nil)
		rr := httptest.NewRecorder()

		mux.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Fatalf("статус = %d, ожидали %d", rr.Code, http.StatusNotFound)
		}
		if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
			t.Fatalf("Content-Type = %q, ожидали application/json", ct)
		}
		var payload errorResponse
		if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
			t.Fatalf("разбор JSON: %v", err)
		}
		if payload.Error == "" {
			t.Fatal("ожидали текст ошибки")
		}
	})

	t.Run("пустой путь", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("статус = %d, ожидали %d", rr.Code, http.StatusNotFound)
		}
	})

	t.Run("вложенный путь", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/a/b", nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("статус = %d, ожидали %d", rr.Code, http.StatusNotFound)
		}
	})

	t.Run("метод не GET", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/"+shortID, nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)
		if rr.Code != http.StatusMethodNotAllowed {
			t.Fatalf("статус = %d, ожидали %d", rr.Code, http.StatusMethodNotAllowed)
		}
	})
}

func TestShortenHandler_GenerateFailure(t *testing.T) {
	orig := generateID
	t.Cleanup(func() { generateID = orig })
	generateID = func() string { return "same" }

	us := NewURLShortener()
	us.urls["same"] = "https://already.example"
	mux := newMux(us)

	req := httptest.NewRequest(http.MethodPost, "/shorten", strings.NewReader(`{"url":"https://example.com/new"}`))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("статус = %d, ожидали %d, тело = %s", rr.Code, http.StatusInternalServerError, rr.Body.String())
	}
}

func TestShortenThenRedirect(t *testing.T) {
	mux := newMux(NewURLShortener())
	const original = "http://example.com/from-handler"

	req := httptest.NewRequest(http.MethodPost, "/shorten", strings.NewReader(`{"url":"`+original+`"}`))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("POST статус = %d, тело = %s", rr.Code, rr.Body.String())
	}

	var created shortenResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &created); err != nil {
		t.Fatalf("разбор JSON: %v", err)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/"+created.ShortURL, nil)
	getRR := httptest.NewRecorder()
	mux.ServeHTTP(getRR, getReq)
	if getRR.Code != http.StatusFound {
		t.Fatalf("GET статус = %d, ожидали 302", getRR.Code)
	}
	if loc := getRR.Header().Get("Location"); loc != original {
		t.Fatalf("Location = %q, ожидали %q", loc, original)
	}
}
