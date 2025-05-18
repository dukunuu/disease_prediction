// server/handlers_symptoms.go
package server

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	// "strconv" // Needed if parseInt32Param is defined here

	"github.com/dukunuu/munkhjin-diplom/backend/db" // Adjust import path
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	// "github.com/jackc/pgx/v5/pgconn" // Import if checking unique constraint errors
)

// --- Request/Response Structs for Symptoms ---

// swagger:model CreateSymptomRequest
type CreateSymptomRequest struct {
	SymptomName        string  `json:"symptom_name"` // Changed to string (required)
	SymptomDescription *string `json:"symptom_description,omitempty"`
}

// swagger:model UpdateSymptomRequest
type UpdateSymptomRequest struct {
	SymptomName        string  `json:"symptom_name"` // Changed to string (required)
	SymptomDescription *string `json:"symptom_description,omitempty"`
}

// swagger:model SymptomResponse
type SymptomResponse struct {
	SymptomID          int32            `json:"symptom_id"`
	SymptomName        string           `json:"symptom_name"` // Changed to string (cannot be null)
	SymptomDescription *string          `json:"symptom_description"`
	CreatedAt          pgtype.Timestamp `json:"created_at"`
	UpdatedAt          pgtype.Timestamp `json:"updated_at"`
}

// --- Assume Helper functions exist (pgtypeText, stringPtrFromPgtypeText, parseInt32Param, etc.) ---
// --- If not, define them here or in a utils package ---

// handleListSymptoms godoc
// @Summary      List symptoms
// @Description  Get a list of all symptoms
// @Tags         Symptoms
// @Accept       json
// @Produce      json
// @Success      200 {array}   SymptomResponse "Successfully retrieved list of symptoms"
// @Failure      500 {object}  HTTPError "Internal server error"
// @Router       /symptoms [get]
func (s *Server) handleListSymptoms() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		symptoms, err := s.queries.ListSymptoms(r.Context())
		if err != nil {
			log.Printf("Error listing symptoms: %v", err)
			respondWithError(w, http.StatusInternalServerError, "Failed to retrieve symptoms")
			return
		}

		responseSymptoms := make([]SymptomResponse, len(symptoms))
		for i, sym := range symptoms {
			// SymptomName from DB cannot be null based on schema and sqlc generation
			responseSymptoms[i] = SymptomResponse{
				SymptomID:          sym.SymptomID,
				SymptomName:        sym.SymptomName, // Directly use string
				SymptomDescription: stringPtrFromPgtypeText(sym.SymptomDescription),
				CreatedAt:          sym.CreatedAt,
				UpdatedAt:          sym.UpdatedAt,
			}
		}
		respondWithJSON(w, http.StatusOK, responseSymptoms)
	}
}

// handleCreateSymptom godoc
// @Summary      Create a new symptom
// @Description  Add a new symptom record. Symptom name is required.
// @Tags         Symptoms
// @Accept       json
// @Produce      json
// @Param        symptom body      CreateSymptomRequest true "Symptom data to create"
// @Success      201     {object}  SymptomResponse "Symptom created successfully"
// @Failure      400     {object}  HTTPError "Invalid request payload or missing symptom name"
// @Failure      409     {object}  HTTPError "Symptom name already exists (unique constraint)"
// @Failure      500     {object}  HTTPError "Internal server error"
// @Router       /symptoms [post]
func (s *Server) handleCreateSymptom() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req CreateSymptomRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid request payload: "+err.Error())
			return
		}
		defer r.Body.Close()

		// Validation: SymptomName is required
		if req.SymptomName == "" {
			respondWithError(w, http.StatusBadRequest, "Missing required field: symptom_name")
			return
		}

		params := db.CreateSymptomParams{
			SymptomName:        req.SymptomName, // Use string directly
			SymptomDescription: pgtypeText(req.SymptomDescription),
		}

		newSymptom, err := s.queries.CreateSymptom(r.Context(), params)
		if err != nil {
			// Optional: Check for unique constraint violation
			// var pgErr *pgconn.PgError
			// if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			// 	respondWithError(w, http.StatusConflict, "Symptom name already exists")
			// 	return
			// }
			log.Printf("Error creating symptom: %v", err)
			respondWithError(w, http.StatusInternalServerError, "Failed to create symptom")
			return
		}

		responseSymptom := SymptomResponse{
			SymptomID:          newSymptom.SymptomID,
			SymptomName:        newSymptom.SymptomName, // Use string directly
			SymptomDescription: stringPtrFromPgtypeText(newSymptom.SymptomDescription),
			CreatedAt:          newSymptom.CreatedAt,
			UpdatedAt:          newSymptom.UpdatedAt,
		}
		respondWithJSON(w, http.StatusCreated, responseSymptom)
	}
}

// handleGetSymptomByID godoc
// @Summary      Get symptom by ID
// @Description  Retrieve details of a specific symptom by its ID
// @Tags         Symptoms
// @Accept       json
// @Produce      json
// @Param        symptomID path      int true "Symptom ID" Format(int32)
// @Success      200       {object}  SymptomResponse "Successfully retrieved symptom"
// @Failure      400       {object}  HTTPError "Invalid Symptom ID format"
// @Failure      404       {object}  HTTPError "Symptom not found"
// @Failure      500       {object}  HTTPError "Internal server error"
// @Router       /symptoms/{symptomID} [get]
func (s *Server) handleGetSymptomByID() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		symptomID, err := parseInt32Param(r, "symptomID")
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid symptom ID: "+err.Error())
			return
		}

		symptom, err := s.queries.GetSymptomByID(r.Context(), symptomID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows) {
				respondWithError(w, http.StatusNotFound, "Symptom not found")
			} else {
				log.Printf("Error retrieving symptom %d: %v", symptomID, err)
				respondWithError(w, http.StatusInternalServerError, "Failed to retrieve symptom")
			}
			return
		}

		responseSymptom := SymptomResponse{
			SymptomID:          symptom.SymptomID,
			SymptomName:        symptom.SymptomName, // Use string directly
			SymptomDescription: stringPtrFromPgtypeText(symptom.SymptomDescription),
			CreatedAt:          symptom.CreatedAt,
			UpdatedAt:          symptom.UpdatedAt,
		}
		respondWithJSON(w, http.StatusOK, responseSymptom)
	}
}

// handleUpdateSymptom godoc
// @Summary      Update symptom details
// @Description  Update details for an existing symptom. Symptom name is required.
// @Tags         Symptoms
// @Accept       json
// @Produce      json
// @Param        symptomID path      int                true "Symptom ID" Format(int32)
// @Param        symptom   body      UpdateSymptomRequest true "Symptom data to update"
// @Success      200       {object}  SymptomResponse "Symptom updated successfully"
// @Failure      400       {object}  HTTPError "Invalid request payload or missing symptom name"
// @Failure      404       {object}  HTTPError "Symptom not found"
// @Failure      409       {object}  HTTPError "Symptom name already exists (unique constraint)"
// @Failure      500       {object}  HTTPError "Internal server error"
// @Router       /symptoms/{symptomID} [put]
func (s *Server) handleUpdateSymptom() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		symptomID, err := parseInt32Param(r, "symptomID")
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid symptom ID: "+err.Error())
			return
		}

		var req UpdateSymptomRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid request payload: "+err.Error())
			return
		}
		defer r.Body.Close()

		// Validation: SymptomName is required
		if req.SymptomName == "" {
			respondWithError(w, http.StatusBadRequest, "Missing required field: symptom_name")
			return
		}

		params := db.UpdateSymptomParams{
			SymptomID:          symptomID,
			SymptomName:        req.SymptomName, // Use string directly
			SymptomDescription: pgtypeText(req.SymptomDescription),
		}

		updatedSymptom, err := s.queries.UpdateSymptom(r.Context(), params)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows) {
				respondWithError(w, http.StatusNotFound, "Symptom not found")
			} else {
				// Optional: Check for unique constraint violation
				// var pgErr *pgconn.PgError
				// if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
				// 	respondWithError(w, http.StatusConflict, "Symptom name already exists")
				// 	return
				// }
				log.Printf("Error updating symptom %d: %v", symptomID, err)
				respondWithError(w, http.StatusInternalServerError, "Failed to update symptom")
			}
			return
		}

		responseSymptom := SymptomResponse{
			SymptomID:          updatedSymptom.SymptomID,
			SymptomName:        updatedSymptom.SymptomName, // Use string directly
			SymptomDescription: stringPtrFromPgtypeText(updatedSymptom.SymptomDescription),
			CreatedAt:          updatedSymptom.CreatedAt,
			UpdatedAt:          updatedSymptom.UpdatedAt,
		}
		respondWithJSON(w, http.StatusOK, responseSymptom)
	}
}

// handleDeleteSymptom godoc
// @Summary      Delete symptom
// @Description  Delete a symptom record by ID. Associated records in junction tables (patient_symptoms, patient_disease_symptom) should be removed via ON DELETE CASCADE.
// @Tags         Symptoms
// @Accept       json
// @Produce      json
// @Param        symptomID path      int true "Symptom ID" Format(int32)
// @Success      204       {string}  string "No Content (Successful deletion)"
// @Failure      400       {object}  HTTPError "Invalid Symptom ID format"
// @Failure      500       {object}  HTTPError "Internal server error"
// @Router       /symptoms/{symptomID} [delete]
func (s *Server) handleDeleteSymptom() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		symptomID, err := parseInt32Param(r, "symptomID")
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid symptom ID: "+err.Error())
			return
		}

		err = s.queries.DeleteSymptom(r.Context(), symptomID)
		if err != nil {
			// Note: DELETE often doesn't error if the ID doesn't exist.
			// FK errors could occur if CASCADE isn't set up correctly.
			log.Printf("Error deleting symptom %d: %v", symptomID, err)
			respondWithError(w, http.StatusInternalServerError, "Failed to delete symptom")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

