package server

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// swagger:type object
type ArbitraryJSON json.RawMessage

// swagger:model HTTPError
type HTTPError struct {
	Error string `json:"error" example:"Resource not found"` // Error message
}

func respondWithJSON(w http.ResponseWriter, code int, payload any) {
	response, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling JSON: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Internal server error marshalling response"}`))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response) // Use Write directly
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	log.Printf("HTTP Error %d: %s", code, message) // Log the error server-side
	respondWithJSON(w, code, map[string]string{"error": message})
}

func parseInt32Param(r *http.Request, paramName string) (int32, error) {
	paramStr := chi.URLParam(r, paramName)
	if paramStr == "" {
		return 0, errors.New("missing URL parameter: " + paramName)
	}
	paramInt, err := strconv.ParseInt(paramStr, 10, 32)
	if err != nil {
		return 0, errors.New("invalid URL parameter format: " + paramName)
	}
	return int32(paramInt), nil
}

func pgtypeText(s *string) pgtype.Text {
	if s == nil || *s == "" {
		return pgtype.Text{} // Represents NULL
	}
	return pgtype.Text{String: *s, Valid: true}
}

func stringPtrFromPgtypeText(pt pgtype.Text) *string {
	if !pt.Valid {
		return nil
	}
	s := pt.String
	return &s
}

func stringFromPgtypeDate(pd pgtype.Date) string {
	if !pd.Valid {
		return ""
	}
	var t time.Time
	err := pd.Scan(&t) // Scan into time.Time
	if err != nil {
		log.Printf("Warning: Failed to scan pgtype.Date back to time.Time: %v", err)
		return ""
	}
	return t.Format("2006-01-02") // Format as YYYY-MM-DD
}

func stringPtrFromPgtypeDate(pd pgtype.Date) *string {
	s := stringFromPgtypeDate(pd)
	if s == "" {
		return nil
	}
	return &s
}

