// server/handlers_relationships.go
package server

import (
	"encoding/json"
	"log"
	"net/http"
	"time" // Needed for diagnosis_date

	"github.com/dukunuu/munkhjin-diplom/backend/db"
	"github.com/jackc/pgx/v5/pgtype" // Needed for pgtype.Date, pgtype.Text etc.
)

// swagger:model RecordPatientSymptomRequest Used for adding general symptoms
type RecordPatientSymptomRequest struct {
	SymptomID    int32      `json:"symptom_id"`
	ReportedDate *time.Time `json:"reported_date,omitempty"` // Use pointer for optional date
}

// swagger:model RecordPatientDiseaseInstanceRequest
type RecordPatientDiseaseInstanceRequest struct {
	DiseaseID     int32      `json:"disease_id"`
	DiagnosisDate *time.Time `json:"diagnosis_date,omitempty"` // Use pointer for optional date
	Notes         *string    `json:"notes,omitempty"`          // Use pointer for optional notes
}

// swagger:model LinkSymptomToDiseaseInstanceRequest
type LinkSymptomToDiseaseInstanceRequest struct {
	SymptomID int32 `json:"symptom_id"`
}

// swagger:model PatientDiseaseInstanceResponse
type PatientDiseaseInstanceResponse struct {
	PatientDiseaseID int32            `json:"patient_disease_id"`
	PatientID        int32            `json:"patient_id"` // Added for context
	DiseaseID        int32            `json:"disease_id"`
	DiseaseName      string           `json:"disease_name"` // Added from join
	DiseaseCode      string           `json:"disease_code"` // Added from join
	DiagnosisDate    pgtype.Date      `json:"diagnosis_date"`
	Notes            *string          `json:"notes"` // Keep as pointer
	CreatedAt        pgtype.Timestamp `json:"created_at"`
	UpdatedAt        pgtype.Timestamp `json:"updated_at"`
}

// --- Helper Functions ---

// Helper to convert *time.Time to pgtype.Date
func pgDateFromTimePtr(t *time.Time) pgtype.Date {
	if t == nil {
		return pgtype.Date{Valid: false}
	}
	return pgtype.Date{Time: *t, Valid: true}
}

// Helper to convert *string to pgtype.Text
func pgTextFromStringPtr(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *s, Valid: true}
}

// === General Patient Symptoms (patient_symptoms table) ===

// handleListGeneralSymptomsForPatient godoc
// @Summary      List general symptoms reported by a patient
// @Description  Get all symptoms associated with a given patient ID via the patient_symptoms table (not linked to specific diagnoses).
// @Tags         Patient Relationships
// @Accept       json
// @Produce      json
// @Param        patientID path      int true "Patient ID" Format(int32)
// @Success      200       {array}   db.ListGeneralSymptomsForPatientRow "Successfully retrieved general symptoms for patient"
// @Failure      400       {object}  HTTPError "Invalid Patient ID format"
// @Failure      500       {object}  HTTPError "Internal server error"
// @Router       /patients/{patientID}/general-symptoms [get]
func (s *Server) handleListGeneralSymptomsForPatient() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		patientID, err := parseInt32Param(r, "patientID")
		if err != nil {
			respondWithError(w, http.StatusBadRequest, err.Error())
			return
		}

		// Use the correct sqlc generated query name
		symptoms, err := s.queries.ListGeneralSymptomsForPatient(r.Context(), patientID)
		if err != nil {
			log.Printf("Error listing general symptoms for patient %d: %v", patientID, err)
			respondWithError(w, http.StatusInternalServerError, "Failed to list general symptoms for patient")
			return
		}

		// The sqlc generated type ListGeneralSymptomsForPatientRow might be suitable directly
		// If not, map it to a specific response struct like SymptomResponse
		// For simplicity, let's assume the generated type is okay for now.
		// If you need SymptomResponse, map it like before.
		respondWithJSON(w, http.StatusOK, symptoms)
	}
}

// handleRecordPatientSymptom godoc
// @Summary      Record a general symptom for a patient
// @Description  Associate an existing symptom with a patient (not tied to a specific diagnosis instance). Optionally include the date it was reported.
// @Tags         Patient Relationships
// @Accept       json
// @Produce      json
// @Param        patientID path      int                          true "Patient ID" Format(int32)
// @Param        symptom   body      RecordPatientSymptomRequest true "Symptom ID and optional reported date"
// @Success      201       {object}  db.PatientSymptom "Symptom recorded successfully"
// @Failure      400       {object}  HTTPError "Invalid Patient ID or request payload"
// @Failure      404       {object}  HTTPError "Patient or Symptom not found (FK constraint)"
// @Failure      409       {object}  HTTPError "Relationship already exists for this date (unique constraint)"
// @Failure      500       {object}  HTTPError "Internal server error"
// @Router       /patients/{patientID}/general-symptoms [post]
func (s *Server) handleRecordPatientSymptom() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		patientID, err := parseInt32Param(r, "patientID")
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid patient ID: "+err.Error())
			return
		}

		var req RecordPatientSymptomRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid request payload: "+err.Error())
			return
		}
		defer r.Body.Close()

		if req.SymptomID <= 0 {
			respondWithError(w, http.StatusBadRequest, "Missing or invalid symptom_id")
			return
		}

		params := db.RecordPatientSymptomParams{
			PatientID:    patientID,
			SymptomID:    req.SymptomID,
			ReportedDate: pgDateFromTimePtr(req.ReportedDate), // Convert *time.Time to pgtype.Date
		}

		// Use the correct sqlc generated query name
		recordedSymptom, err := s.queries.RecordPatientSymptom(r.Context(), params)
		if err != nil {
			// Add more specific error checking if needed (e.g., for 409 Conflict)
			// var pgErr *pgconn.PgError
			// if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			// 	respondWithError(w, http.StatusConflict, "Symptom already recorded for this patient on this date")
			// 	return
			// }
			log.Printf("Error recording symptom %d for patient %d: %v", req.SymptomID, patientID, err)
			respondWithError(w, http.StatusInternalServerError, "Failed to record symptom for patient")
			return
		}

		respondWithJSON(w, http.StatusCreated, recordedSymptom)
	}
}

// handleDeletePatientSymptom godoc
// @Summary      Delete a specific general symptom record
// @Description  Remove a specific patient_symptoms entry by its unique ID.
// @Tags         Patient Relationships
// @Accept       json
// @Produce      json
// @Param        id path      int true "Patient Symptom Record ID" Format(int32)
// @Success      204       {string}  string "No Content (Successful removal)"
// @Failure      400       {object}  HTTPError "Invalid ID format"
// @Failure      500       {object}  HTTPError "Internal server error"
// @Router       /patient-symptoms/{id} [delete]
func (s *Server) handleDeletePatientSymptom() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		patientSymptomID, err := parseInt32Param(r, "id") // Get ID from path
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid patient symptom record ID: "+err.Error())
			return
		}

		// Use the correct sqlc generated query name
		err = s.queries.RemovePatientSymptomByID(r.Context(), patientSymptomID)
		if err != nil {
			// DELETE might not error if the record doesn't exist. Check if needed.
			log.Printf("Error removing patient symptom record %d: %v", patientSymptomID, err)
			respondWithError(w, http.StatusInternalServerError, "Failed to remove patient symptom record")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// === Patient Disease Instances (patient_disease table) ===

// handleListDiseaseInstancesForPatient godoc
// @Summary      List disease instances for a specific patient
// @Description  Get all recorded disease instances (diagnoses) for a given patient ID.
// @Tags         Patient Relationships
// @Accept       json
// @Produce      json
// @Param        patientID path      int true "Patient ID" Format(int32)
// @Success      200       {array}   PatientDiseaseInstanceResponse "Successfully retrieved disease instances"
// @Failure      400       {object}  HTTPError "Invalid Patient ID format"
// @Failure      500       {object}  HTTPError "Internal server error"
// @Router       /patients/{patientID}/disease-instances [get]
func (s *Server) handleListDiseaseInstancesForPatient() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		patientID, err := parseInt32Param(r, "patientID")
		if err != nil {
			respondWithError(w, http.StatusBadRequest, err.Error())
			return
		}

		// Use the correct sqlc generated query name
		instances, err := s.queries.ListDiseaseInstancesForPatient(r.Context(), patientID)
		if err != nil {
			log.Printf("Error listing disease instances for patient %d: %v", patientID, err)
			respondWithError(w, http.StatusInternalServerError, "Failed to list disease instances for patient")
			return
		}

		// Map the result to the response struct
		responseInstances := make([]PatientDiseaseInstanceResponse, len(instances))
		for i, inst := range instances {
			responseInstances[i] = PatientDiseaseInstanceResponse{
				PatientDiseaseID: inst.PatientDiseaseID,
				PatientID:        patientID, // Add patientID from context
				DiseaseID:        inst.DiseaseID,
				DiseaseName:      inst.DiseaseName,
				DiseaseCode:      inst.DiseaseCode,
				DiagnosisDate:    inst.DiagnosisDate,
				Notes:            stringPtrFromPgtypeText(inst.Notes), // Convert pgtype.Text to *string
				CreatedAt:        inst.CreatedAt,
				UpdatedAt:        inst.UpdatedAt,
			}
		}

		respondWithJSON(w, http.StatusOK, responseInstances)
	}
}

// handleRecordPatientDiseaseInstance godoc
// @Summary      Record a disease instance for a patient
// @Description  Create a record of a specific disease diagnosis for a patient, with optional date and notes.
// @Tags         Patient Relationships
// @Accept       json
// @Produce      json
// @Param        patientID path      int                                   true "Patient ID" Format(int32)
// @Param        instance  body      RecordPatientDiseaseInstanceRequest true "Disease ID, optional diagnosis date and notes"
// @Success      201       {object}  db.PatientDisease "Disease instance recorded successfully"
// @Failure      400       {object}  HTTPError "Invalid Patient ID or request payload"
// @Failure      404       {object}  HTTPError "Patient or Disease not found (FK constraint)"
// @Failure      409       {object}  HTTPError "Duplicate instance for this patient/disease/date (unique constraint)"
// @Failure      500       {object}  HTTPError "Internal server error"
// @Router       /patients/{patientID}/disease-instances [post]
func (s *Server) handleRecordPatientDiseaseInstance() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		patientID, err := parseInt32Param(r, "patientID")
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid patient ID: "+err.Error())
			return
		}

		var req RecordPatientDiseaseInstanceRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid request payload: "+err.Error())
			return
		}
		defer r.Body.Close()

		if req.DiseaseID <= 0 {
			respondWithError(w, http.StatusBadRequest, "Missing or invalid disease_id")
			return
		}

		params := db.RecordPatientDiseaseInstanceParams{
			PatientID:     patientID,
			DiseaseID:     req.DiseaseID,
			DiagnosisDate: pgDateFromTimePtr(req.DiagnosisDate), // Convert *time.Time to pgtype.Date
			Notes:         pgTextFromStringPtr(req.Notes),       // Convert *string to pgtype.Text
		}

		// Use the correct sqlc generated query name
		instance, err := s.queries.RecordPatientDiseaseInstance(r.Context(), params)
		if err != nil {
			// Add specific error checking (e.g., 409 Conflict) if needed
			log.Printf("Error recording disease instance for patient %d, disease %d: %v", patientID, req.DiseaseID, err)
			respondWithError(w, http.StatusInternalServerError, "Failed to record disease instance")
			return
		}

		respondWithJSON(w, http.StatusCreated, instance)
	}
}

// handleDeletePatientDiseaseInstance godoc
// @Summary      Delete a specific disease instance record
// @Description  Remove a specific patient_disease entry by its unique ID (patient_disease_id). This also removes associated symptom links via ON DELETE CASCADE.
// @Tags         Patient Relationships
// @Accept       json
// @Produce      json
// @Param        instanceID path      int true "Patient Disease Instance ID" Format(int32)
// @Success      204       {string}  string "No Content (Successful removal)"
// @Failure      400       {object}  HTTPError "Invalid ID format"
// @Failure      500       {object}  HTTPError "Internal server error"
// @Router       /disease-instances/{instanceID} [delete]
func (s *Server) handleDeletePatientDiseaseInstance() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		instanceID, err := parseInt32Param(r, "instanceID") // Get ID from path
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid disease instance ID: "+err.Error())
			return
		}

		// Use the correct sqlc generated query name
		err = s.queries.DeletePatientDiseaseInstance(r.Context(), instanceID)
		if err != nil {
			// DELETE might not error if the record doesn't exist.
			log.Printf("Error removing disease instance %d: %v", instanceID, err)
			respondWithError(w, http.StatusInternalServerError, "Failed to remove disease instance")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// === Disease Instance Symptom Linking (patient_disease_symptom table) ===

// handleGetSymptomsForDiseaseInstance godoc
// @Summary      List symptoms for a specific disease instance
// @Description  Get all symptoms linked to a specific patient_disease record (diagnosis instance).
// @Tags         Patient Relationships
// @Accept       json
// @Produce      json
// @Param        instanceID path      int true "Patient Disease Instance ID" Format(int32)
// @Success      200       {array}   db.GetSymptomsForPatientDiseaseInstanceRow "Successfully retrieved symptoms for the instance"
// @Failure      400       {object}  HTTPError "Invalid Instance ID format"
// @Failure      404       {object}  HTTPError "Disease instance not found"
// @Failure      500       {object}  HTTPError "Internal server error"
// @Router       /disease-instances/{instanceID}/symptoms [get]
func (s *Server) handleGetSymptomsForDiseaseInstance() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		instanceID, err := parseInt32Param(r, "instanceID")
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid disease instance ID: "+err.Error())
			return
		}

		// Use the correct sqlc generated query name
		symptoms, err := s.queries.GetSymptomsForPatientDiseaseInstance(r.Context(), instanceID)
		if err != nil {
			// Check if the instance itself was not found (though the query might just return empty)
			log.Printf("Error listing symptoms for disease instance %d: %v", instanceID, err)
			respondWithError(w, http.StatusInternalServerError, "Failed to list symptoms for disease instance")
			return
		}

		// The sqlc generated type GetSymptomsForPatientDiseaseInstanceRow might be suitable directly.
		// If not, map it to SymptomResponse.
		respondWithJSON(w, http.StatusOK, symptoms)
	}
}

// handleLinkSymptomToDiseaseInstance godoc
// @Summary      Link a symptom to a disease instance
// @Description  Create an association between an existing symptom and a specific patient disease instance.
// @Tags         Patient Relationships
// @Accept       json
// @Produce      json
// @Param        instanceID path      int                                   true "Patient Disease Instance ID" Format(int32)
// @Param        link       body      LinkSymptomToDiseaseInstanceRequest true "Symptom ID to link"
// @Success      201       {object}  db.PatientDiseaseSymptom "Symptom linked successfully"
// @Failure      400       {object}  HTTPError "Invalid Instance ID or request payload"
// @Failure      404       {object}  HTTPError "Disease instance or Symptom not found (FK constraint)"
// @Failure      409       {object}  HTTPError "Symptom already linked to this instance (unique constraint)"
// @Failure      500       {object}  HTTPError "Internal server error"
// @Router       /disease-instances/{instanceID}/symptoms [post]
func (s *Server) handleLinkSymptomToDiseaseInstance() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		instanceID, err := parseInt32Param(r, "instanceID")
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid disease instance ID: "+err.Error())
			return
		}

		var req LinkSymptomToDiseaseInstanceRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid request payload: "+err.Error())
			return
		}
		defer r.Body.Close()

		if req.SymptomID <= 0 {
			respondWithError(w, http.StatusBadRequest, "Missing or invalid symptom_id")
			return
		}

		params := db.LinkSymptomToPatientDiseaseParams{
			PatientDiseaseID: instanceID,
			SymptomID:        req.SymptomID,
		}

		// Use the correct sqlc generated query name
		link, err := s.queries.LinkSymptomToPatientDisease(r.Context(), params)
		if err != nil {
			// Add specific error checking (e.g., 409 Conflict) if needed
			log.Printf("Error linking symptom %d to disease instance %d: %v", req.SymptomID, instanceID, err)
			respondWithError(w, http.StatusInternalServerError, "Failed to link symptom to disease instance")
			return
		}

		respondWithJSON(w, http.StatusCreated, link)
	}
}

// handleUnlinkSymptomFromDiseaseInstance godoc
// @Summary      Unlink a symptom from a disease instance
// @Description  Remove the association between a symptom and a specific patient disease instance.
// @Tags         Patient Relationships
// @Accept       json
// @Produce      json
// @Param        instanceID path      int true "Patient Disease Instance ID" Format(int32)
// @Param        symptomID  path      int true "Symptom ID to unlink" Format(int32)
// @Success      204       {string}  string "No Content (Successful removal)"
// @Failure      400       {object}  HTTPError "Invalid Instance ID or Symptom ID format"
// @Failure      500       {object}  HTTPError "Internal server error"
// @Router       /disease-instances/{instanceID}/symptoms/{symptomID} [delete]
func (s *Server) handleUnlinkSymptomFromDiseaseInstance() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		instanceID, err := parseInt32Param(r, "instanceID")
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid disease instance ID: "+err.Error())
			return
		}
		symptomID, err := parseInt32Param(r, "symptomID")
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid symptom ID: "+err.Error())
			return
		}

		params := db.UnlinkSymptomFromPatientDiseaseParams{
			PatientDiseaseID: instanceID,
			SymptomID:        symptomID,
		}

		// Use the correct sqlc generated query name
		err = s.queries.UnlinkSymptomFromPatientDisease(r.Context(), params)
		if err != nil {
			// DELETE might not error if the link doesn't exist.
			log.Printf("Error unlinking symptom %d from disease instance %d: %v", symptomID, instanceID, err)
			respondWithError(w, http.StatusInternalServerError, "Failed to unlink symptom from disease instance")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
