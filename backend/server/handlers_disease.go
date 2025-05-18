package server

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/dukunuu/munkhjin-diplom/backend/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// swagger:model CreateDiseaseRequest
type CreateDiseaseRequest struct {
	DiseaseName        string          `json:"disease_name"`
	DiseaseCode        string          `json:"disease_code"`
	DiseaseDescription *string         `json:"disease_description,omitempty"`
	DiseaseTreatment   ArbitraryJSON 	 `json:"disease_treatment,omitempty"` // Accept raw JSON
}

// swagger:model UpdateDiseaseRequest
type UpdateDiseaseRequest struct {
	DiseaseName        string          `json:"disease_name"`
	DiseaseCode        string          `json:"disease_code"`
	DiseaseDescription *string         `json:"disease_description,omitempty"`
	DiseaseTreatment   ArbitraryJSON   `json:"disease_treatment,omitempty"` // Accept raw JSON
}

// swagger:model DiseaseResponse
type DiseaseResponse struct {
	DiseaseID          int32           `json:"disease_id"`
	DiseaseName        string          `json:"disease_name"`
	DiseaseCode        string          `json:"disease_code"`
	DiseaseDescription *string         `json:"disease_description"`
	DiseaseTreatment   ArbitraryJSON   `json:"disease_treatment"` // Send raw JSON back
	CreatedAt          pgtype.Timestamp `json:"created_at"` // Or format as string
	UpdatedAt          pgtype.Timestamp `json:"updated_at"` // Or format as string
}

// handleListDiseases godoc
// @Summary      List diseases
// @Description  Get a list of all diseases
// @Tags         Diseases
// @Accept       json
// @Produce      json
// @Success      200 {array}   DiseaseResponse "Successfully retrieved list of diseases"
// @Failure      500 {object}  HTTPError "Internal server error"
// @Router       /diseases [get]
func (s *Server) handleListDiseases() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		diseases, err := s.queries.ListDiseases(r.Context())
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to retrieve diseases")
			return
		}
		responseDiseases := make([]DiseaseResponse, len(diseases))
		for i, d := range diseases {
			responseDiseases[i] = DiseaseResponse{
				DiseaseID:          d.DiseaseID,
				DiseaseName:        d.DiseaseName,
				DiseaseCode:        d.DiseaseCode,
				DiseaseDescription: stringPtrFromPgtypeText(d.DiseaseDescription),
				DiseaseTreatment:   d.DiseaseTreatment, // Pass raw bytes
				CreatedAt:          d.CreatedAt,
				UpdatedAt:          d.UpdatedAt,
			}
		}
		respondWithJSON(w, http.StatusOK, responseDiseases)
	}
}

// handleCreateDisease godoc
// @Summary      Create a new disease
// @Description  Add a new disease record
// @Tags         Diseases
// @Accept       json
// @Produce      json
// @Param        disease body      CreateDiseaseRequest true "Disease data to create"
// @Success      201     {object}  DiseaseResponse "Disease created successfully"
// @Failure      400     {object}  HTTPError "Invalid request payload or validation error"
// @Failure      500     {object}  HTTPError "Internal server error"
// @Router       /diseases [post]
func (s *Server) handleCreateDisease() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req CreateDiseaseRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid request payload: "+err.Error())
			return
		}
		defer r.Body.Close()

		if req.DiseaseName == "" || req.DiseaseCode == "" {
			respondWithError(w, http.StatusBadRequest, "Missing required fields: disease_name, disease_code")
			return
		}

		// Ensure treatment is valid JSON or null before sending to DB
		var treatmentBytes []byte
		if len(req.DiseaseTreatment) > 0 {
			if !json.Valid(req.DiseaseTreatment) {
				respondWithError(w, http.StatusBadRequest, "Invalid JSON format for disease_treatment")
				return
			}
			treatmentBytes = req.DiseaseTreatment
		} // If empty or null, treatmentBytes remains nil, which DB handles as NULL

		params := db.CreateDiseaseParams{
			DiseaseName:        req.DiseaseName,
			DiseaseCode:        req.DiseaseCode,
			DiseaseDescription: pgtypeText(req.DiseaseDescription),
			DiseaseTreatment:   treatmentBytes,
		}

		newDisease, err := s.queries.CreateDisease(r.Context(), params)
		if err != nil {
			// TODO: Check unique constraints
			log.Printf("Error creating disease: %v", err)
			respondWithError(w, http.StatusInternalServerError, "Failed to create disease")
			return
		}

		responseDisease := DiseaseResponse{
			DiseaseID:          newDisease.DiseaseID,
			DiseaseName:        newDisease.DiseaseName,
			DiseaseCode:        newDisease.DiseaseCode,
			DiseaseDescription: stringPtrFromPgtypeText(newDisease.DiseaseDescription),
			DiseaseTreatment:   newDisease.DiseaseTreatment,
			CreatedAt:          newDisease.CreatedAt,
			UpdatedAt:          newDisease.UpdatedAt,
		}
		respondWithJSON(w, http.StatusCreated, responseDisease)
	}
}

// handleGetDiseaseByID godoc
// @Summary      Get disease by ID
// @Description  Retrieve details of a specific disease by its ID
// @Tags         Diseases
// @Accept       json
// @Produce      json
// @Param        diseaseID path      int true "Disease ID" Format(int32)
// @Success      200       {object}  DiseaseResponse "Successfully retrieved disease"
// @Failure      400       {object}  HTTPError "Invalid Disease ID format"
// @Failure      404       {object}  HTTPError "Disease not found"
// @Failure      500       {object}  HTTPError "Internal server error"
// @Router       /diseases/{diseaseID} [get]
func (s *Server) handleGetDiseaseByID() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		diseaseID, err := parseInt32Param(r, "diseaseID")
		if err != nil {
			respondWithError(w, http.StatusBadRequest, err.Error())
			return
		}

		disease, err := s.queries.GetDiseaseByID(r.Context(), diseaseID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows) {
				respondWithError(w, http.StatusNotFound, "Disease not found")
			} else {
				log.Printf("Error retrieving disease %d: %v", diseaseID, err)
				respondWithError(w, http.StatusInternalServerError, "Failed to retrieve disease")
			}
			return
		}

		responseDisease := DiseaseResponse{
			DiseaseID:          disease.DiseaseID,
			DiseaseName:        disease.DiseaseName,
			DiseaseCode:        disease.DiseaseCode,
			DiseaseDescription: stringPtrFromPgtypeText(disease.DiseaseDescription),
			DiseaseTreatment:   disease.DiseaseTreatment,
			CreatedAt:          disease.CreatedAt,
			UpdatedAt:          disease.UpdatedAt,
		}
		respondWithJSON(w, http.StatusOK, responseDisease)
	}
}

// handleUpdateDisease godoc
// @Summary      Update disease details
// @Description  Update details for an existing disease
// @Tags         Diseases
// @Accept       json
// @Produce      json
// @Param        diseaseID path      int                true "Disease ID" Format(int32)
// @Param        disease   body      UpdateDiseaseRequest true "Disease data to update"
// @Success      200       {object}  DiseaseResponse "Disease updated successfully"
// @Failure      400       {object}  HTTPError "Invalid request payload or validation error"
// @Failure      404       {object}  HTTPError "Disease not found"
// @Failure      500       {object}  HTTPError "Internal server error"
// @Router       /diseases/{diseaseID} [put]
func (s *Server) handleUpdateDisease() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		diseaseID, err := parseInt32Param(r, "diseaseID")
		if err != nil {
			respondWithError(w, http.StatusBadRequest, err.Error())
			return
		}

		var req UpdateDiseaseRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid request payload: "+err.Error())
			return
		}
		defer r.Body.Close()

		if req.DiseaseName == "" || req.DiseaseCode == "" {
			respondWithError(w, http.StatusBadRequest, "Missing required fields: disease_name, disease_code")
			return
		}

		var treatmentBytes []byte
		if len(req.DiseaseTreatment) > 0 {
			if !json.Valid(req.DiseaseTreatment) {
				respondWithError(w, http.StatusBadRequest, "Invalid JSON format for disease_treatment")
				return
			}
			treatmentBytes = req.DiseaseTreatment
		}

		params := db.UpdateDiseaseParams{
			DiseaseID:          diseaseID,
			DiseaseName:        req.DiseaseName,
			DiseaseCode:        req.DiseaseCode,
			DiseaseDescription: pgtypeText(req.DiseaseDescription),
			DiseaseTreatment:   treatmentBytes,
		}

		updatedDisease, err := s.queries.UpdateDisease(r.Context(), params)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows) {
				respondWithError(w, http.StatusNotFound, "Disease not found")
			} else {
				// TODO: Check unique constraints
				log.Printf("Error updating disease %d: %v", diseaseID, err)
				respondWithError(w, http.StatusInternalServerError, "Failed to update disease")
			}
			return
		}

		responseDisease := DiseaseResponse{
			DiseaseID:          updatedDisease.DiseaseID,
			DiseaseName:        updatedDisease.DiseaseName,
			DiseaseCode:        updatedDisease.DiseaseCode,
			DiseaseDescription: stringPtrFromPgtypeText(updatedDisease.DiseaseDescription),
			DiseaseTreatment:   updatedDisease.DiseaseTreatment,
			CreatedAt:          updatedDisease.CreatedAt,
			UpdatedAt:          updatedDisease.UpdatedAt,
		}
		respondWithJSON(w, http.StatusOK, responseDisease)
	}
}

// handleDeleteDisease godoc
// @Summary      Delete disease
// @Description  Delete a disease record by ID
// @Tags         Diseases
// @Accept       json
// @Produce      json
// @Param        diseaseID path      int true "Disease ID" Format(int32)
// @Success      204       {string}  string "No Content (Successful deletion)"
// @Failure      400       {object}  HTTPError "Invalid Disease ID format"
// @Failure      500       {object}  HTTPError "Internal server error"
// @Router       /diseases/{diseaseID} [delete]
func (s *Server) handleDeleteDisease() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		diseaseID, err := parseInt32Param(r, "diseaseID")
		if err != nil {
			respondWithError(w, http.StatusBadRequest, err.Error())
			return
		}

		err = s.queries.DeleteDisease(r.Context(), diseaseID)
		if err != nil {
			log.Printf("Error deleting disease %d: %v", diseaseID, err)
			respondWithError(w, http.StatusInternalServerError, "Failed to delete disease")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

