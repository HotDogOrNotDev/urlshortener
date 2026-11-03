package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
)

type shortenRequest struct {
	URL string `json:"url"`
}

type shortenResponse struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func main() {
	if err := run(":8080"); err != nil {
		log.Fatal(err)
	}
}

func run(addr string) error {
	log.Printf("сервер запущен на %s", addr)
	return http.ListenAndServe(addr, newMux(NewURLShortener()))
}

func newMux(shortener *URLShortener) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/shorten", shortenHandler(shortener))
	mux.HandleFunc("/", redirectHandler(shortener))
	return mux
}

func shortenHandler(us *URLShortener) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "метод не поддерживается"})
			return
		}

		var req shortenRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "некорректный JSON"})
			return
		}

		shortID, err := us.Shorten(req.URL)
		if err != nil {
			if errors.Is(err, ErrInvalidURL) {
				writeJSON(w, http.StatusBadRequest, errorResponse{Error: "невалидный URL"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "не удалось сократить URL"})
			return
		}

		writeJSON(w, http.StatusOK, shortenResponse{
			ShortURL:    shortID,
			OriginalURL: req.URL,
		})
	}
}

func redirectHandler(us *URLShortener) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "метод не поддерживается"})
			return
		}

		shortID := strings.Trim(r.URL.Path, "/")
		if shortID == "" || strings.Contains(shortID, "/") {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "короткий URL не найден"})
			return
		}

		original, err := us.GetOriginal(shortID)
		if err != nil {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "короткий URL не найден"})
			return
		}

		http.Redirect(w, r, original, http.StatusFound)
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
