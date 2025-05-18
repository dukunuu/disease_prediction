// server/handlers_patients.go
package server

import (
	"database/sql" // Required for sql.ErrNoRows check alongside pgx.ErrNoRows
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dukunuu/munkhjin-diplom/backend/db" // Adjust import path if needed
	"github.com/jackc/pgx/v5"                     // For pgx.ErrNoRows
	"github.com/jackc/pgx/v5/pgtype"
)

// --- Request/Response Structs for Patients ---

// swagger:model CreatePatientRequest
type CreatePatientRequest struct {
	Firstname   string  `json:"firstname"`
	Lastname    string  `json:"lastname"`
	Register    string  `json:"register"`
	Age         int32   `json:"age"`
	Gender      string  `json:"gender"`
	Birthdate   string  `json:"birthdate"` // Expect YYYY-MM-DD string
	Address     *string `json:"address,omitempty"` // Use pointer for optional field
	Phonenumber string  `json:"phonenumber"`
	Email       string  `json:"email"`
}

// swagger:model UpdatePatientRequest
type UpdatePatientRequest struct {
	Firstname   string  `json:"firstname"`
	Lastname    string  `json:"lastname"`
	Register    string  `json:"register"`
	Age         int32   `json:"age"`
	Gender      string  `json:"gender"`
	Birthdate   string  `json:"birthdate"` // Expect YYYY-MM-DD string
	Address     *string `json:"address,omitempty"`
	Phonenumber string  `json:"phonenumber"`
	Email       string  `json:"email"`
}

// swagger:model PatientResponse
type PatientResponse struct {
	PatientID   int32   `json:"patient_id"`
	Firstname   string  `json:"firstname"`
	Lastname    string  `json:"lastname"`
	Register    string  `json:"register"`
	Age         int32   `json:"age"`
	Gender      string  `json:"gender"`
	Birthdate   *string `json:"birthdate"` // YYYY-MM-DD or null
	Address     *string `json:"address"`   // string or null
	Phonenumber string  `json:"phonenumber"`
	Email       string  `json:"email"`
}

// swagger:model PatientDetailsResponse Used for the /details endpoint with aggregated lists
type PatientDetailsResponse struct {
	PatientID            int32    `json:"patient_id"`
	Firstname            string   `json:"firstname"`
	Lastname             string   `json:"lastname"`
	Email                string   `json:"email"`
	GeneralSymptomsList  []string `json:"general_symptoms_list"`  // Parsed list from patient_symptoms
	DistinctDiseasesList []string `json:"distinct_diseases_list"` // Parsed list from patient_disease
}

// --- Helper Functions for Type Conversion (ensure these are accessible) ---


// Helper to convert YYYY-MM-DD string to pgtype.Date
func pgDateFromString(dateStr string) (pgtype.Date, error) {
	if dateStr == "" {
		return pgtype.Date{Valid: false}, nil // Handle empty string as NULL
	}
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return pgtype.Date{}, err // Invalid format
	}
	return pgtype.Date{Time: t, Valid: true}, nil
}


// Helper to parse comma-separated string from STRING_AGG (byte slice)
func parseAggregatedList(byteData []byte) []string {
	if len(byteData) == 0 {
		return []string{} // Return empty slice, not nil
	}
	strData := string(byteData)
	if strData == "" {
		return []string{}
	}
	// Split and trim whitespace from each item
	items := strings.Split(strData, ",")
	result := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" { // Avoid adding empty strings if there are extra commas
			result = append(result, trimmed)
		}
	}
	return result
}

// Assume parseInt32Param, respondWithError, respondWithJSON exist elsewhere

// --- Patient Handlers ---

// handleListPatients godoc
// @Summary      List patients
// @Description  Get a paginated list of patients
// @Tags         Patients
// @Accept       json
// @Produce      json
// @Param        limit   query     int  false  "Pagination limit" default(10)
// @Param        offset  query     int  false  "Pagination offset" default(0)
// @Success      200     {array}   PatientResponse "Successfully retrieved list of patients"
// @Failure      500     {object}  HTTPError "Internal server error"
// @Router       /patients [get]
func (s *Server) handleListPatients() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limitStr := r.URL.Query().Get("limit")
		offsetStr := r.URL.Query().Get("offset")

		limit, err := strconv.ParseInt(limitStr, 10, 32)
		if err != nil || limit <= 0 {
			limit = 10 // Default limit
		}

		offset, err := strconv.ParseInt(offsetStr, 10, 32)
		if err != nil || offset < 0 {
			offset = 0 // Default offset
		}

		params := db.ListPatientsParams{
			Limit:  int32(limit),
			Offset: int32(offset),
		}

		patients, err := s.queries.ListPatients(r.Context(), params)
		if err != nil {
			log.Printf("Error listing patients: %v", err)
			respondWithError(w, http.StatusInternalServerError, "Failed to retrieve patients")
			return
		}

		// Convert db.Patient to PatientResponse
		responsePatients := make([]PatientResponse, len(patients))
		for i, p := range patients {
			responsePatients[i] = PatientResponse{
				PatientID:   p.PatientID,
				Firstname:   p.Firstname,
				Lastname:    p.Lastname,
				Register:    p.Register,
				Age:         p.Age,
				Gender:      p.Gender,
				Birthdate:   stringPtrFromPgtypeDate(p.Birthdate),
				Address:     stringPtrFromPgtypeText(p.Address),
				Phonenumber: p.Phonenumber,
				Email:       p.Email,
			}
		}

		respondWithJSON(w, http.StatusOK, responsePatients)
	}
}

// handleCreatePatient godoc
// @Summary      Create a new patient
// @Description  Add a new patient record to the database
// @Tags         Patients
// @Accept       json
// @Produce      json
// @Param        patient body      CreatePatientRequest true "Patient data to create"
// @Success      201     {object}  PatientResponse "Patient created successfully"
// @Failure      400     {object}  HTTPError "Invalid request payload or validation error"
// @Failure      500     {object}  HTTPError "Internal server error (e.g., DB error)"
// @Router       /patients [post]
func (s *Server) handleCreatePatient() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req CreatePatientRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid request payload: "+err.Error())
			return
		}
		defer r.Body.Close()

		// Basic Validation
		if req.Firstname == "" || req.Lastname == "" || req.Email == "" || req.Register == "" || req.Phonenumber == "" || req.Gender == "" || req.Birthdate == "" {
			respondWithError(w, http.StatusBadRequest, "Missing required fields (firstname, lastname, email, register, phonenumber, gender, birthdate)")
			return
		}

		birthdatePg, err := pgDateFromString(req.Birthdate)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid birthdate format (use YYYY-MM-DD)")
			return
		}

		params := db.CreatePatientParams{
			Firstname:   req.Firstname,
			Lastname:    req.Lastname,
			Register:    req.Register,
			Age:         req.Age, // Assuming age is provided correctly
			Gender:      req.Gender,
			Birthdate:   birthdatePg,
			Address:     pgtypeText(req.Address), // Use helper for nullable text
			Phonenumber: req.Phonenumber,
			Email:       req.Email,
		}

		newPatient, err := s.queries.CreatePatient(r.Context(), params)
		if err != nil {
			// TODO: Check for specific DB errors like unique constraint violation on email
			log.Printf("Error creating patient: %v", err)
			respondWithError(w, http.StatusInternalServerError, "Failed to create patient")
			return
		}

		// Convert db.Patient to PatientResponse
		responsePatient := PatientResponse{
			PatientID:   newPatient.PatientID,
			Firstname:   newPatient.Firstname,
			Lastname:    newPatient.Lastname,
			Register:    newPatient.Register,
			Age:         newPatient.Age,
			Gender:      newPatient.Gender,
			Birthdate:   stringPtrFromPgtypeDate(newPatient.Birthdate),
			Address:     stringPtrFromPgtypeText(newPatient.Address),
			Phonenumber: newPatient.Phonenumber,
			Email:       newPatient.Email,
		}

		respondWithJSON(w, http.StatusCreated, responsePatient)
	}
}

// handleGetPatientByID godoc
// @Summary      Get patient by ID
// @Description  Retrieve details of a specific patient by their ID
// @Tags         Patients
// @Accept       json
// @Produce      json
// @Param        patientID path      int true "Patient ID" Format(int32)
// @Success      200       {object}  PatientResponse "Successfully retrieved patient"
// @Failure      400       {object}  HTTPError "Invalid Patient ID format"
// @Failure      404       {object}  HTTPError "Patient not found"
// @Failure      500       {object}  HTTPError "Internal server error"
// @Router       /patients/{patientID} [get]
func (s *Server) handleGetPatientByID() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		patientID, err := parseInt32Param(r, "patientID")
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid patient ID: "+err.Error())
			return
		}

		patient, err := s.queries.GetPatientByID(r.Context(), patientID)
		if err != nil {
			// Check for pgx specific no rows error first
			if errors.Is(err, pgx.ErrNoRows) {
				respondWithError(w, http.StatusNotFound, "Patient not found")
			} else if errors.Is(err, sql.ErrNoRows) { // Fallback check (less likely with pgx)
				respondWithError(w, http.StatusNotFound, "Patient not found")
			} else {
				log.Printf("Error retrieving patient %d: %v", patientID, err)
				respondWithError(w, http.StatusInternalServerError, "Failed to retrieve patient")
			}
			return
		}

		// Convert db.Patient to PatientResponse
		responsePatient := PatientResponse{
			PatientID:   patient.PatientID,
			Firstname:   patient.Firstname,
			Lastname:    patient.Lastname,
			Register:    patient.Register,
			Age:         patient.Age,
			Gender:      patient.Gender,
			Birthdate:   stringPtrFromPgtypeDate(patient.Birthdate),
			Address:     stringPtrFromPgtypeText(patient.Address),
			Phonenumber: patient.Phonenumber,
			Email:       patient.Email,
		}

		respondWithJSON(w, http.StatusOK, responsePatient)
	}
}

// handleUpdatePatientDetails godoc
// @Summary      Update patient details
// @Description  Update details for an existing patient
// @Tags         Patients
// @Accept       json
// @Produce      json
// @Param        patientID path      int                true "Patient ID" Format(int32)
// @Param        patient   body      UpdatePatientRequest true "Patient data to update"
// @Success      200       {object}  PatientResponse "Patient updated successfully"
// @Failure      400       {object}  HTTPError "Invalid request payload or validation error"
// @Failure      404       {object}  HTTPError "Patient not found"
// @Failure      500       {object}  HTTPError "Internal server error"
// @Router       /patients/{patientID} [put]
func (s *Server) handleUpdatePatientDetails() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		patientID, err := parseInt32Param(r, "patientID")
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid patient ID: "+err.Error())
			return
		}

		var req UpdatePatientRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid request payload: "+err.Error())
			return
		}
		defer r.Body.Close()

		// Basic Validation
		if req.Firstname == "" || req.Lastname == "" || req.Email == "" || req.Register == "" || req.Phonenumber == "" || req.Gender == "" || req.Birthdate == "" {
			respondWithError(w, http.StatusBadRequest, "Missing required fields (firstname, lastname, email, register, phonenumber, gender, birthdate)")
			return
		}

		birthdatePg, err := pgDateFromString(req.Birthdate)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid birthdate format (use YYYY-MM-DD)")
			return
		}

		params := db.UpdatePatientDetailsParams{
			PatientID:   patientID, // From URL param
			Firstname:   req.Firstname,
			Lastname:    req.Lastname,
			Register:    req.Register,
			Age:         req.Age,
			Gender:      req.Gender,
			Birthdate:   birthdatePg,
			Address:     pgtypeText(req.Address),
			Phonenumber: req.Phonenumber,
			Email:       req.Email,
		}

		updatedPatient, err := s.queries.UpdatePatientDetails(r.Context(), params)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows) {
				respondWithError(w, http.StatusNotFound, "Patient not found")
			} else {
				// TODO: Check for specific DB errors like unique constraints on email update
				log.Printf("Error updating patient %d: %v", patientID, err)
				respondWithError(w, http.StatusInternalServerError, "Failed to update patient")
			}
			return
		}

		// Convert db.Patient to PatientResponse
		responsePatient := PatientResponse{
			PatientID:   updatedPatient.PatientID,
			Firstname:   updatedPatient.Firstname,
			Lastname:    updatedPatient.Lastname,
			Register:    updatedPatient.Register,
			Age:         updatedPatient.Age,
			Gender:      updatedPatient.Gender,
			Birthdate:   stringPtrFromPgtypeDate(updatedPatient.Birthdate),
			Address:     stringPtrFromPgtypeText(updatedPatient.Address),
			Phonenumber: updatedPatient.Phonenumber,
			Email:       updatedPatient.Email,
		}

		respondWithJSON(w, http.StatusOK, responsePatient)
	}
}

// handleDeletePatient godoc
// @Summary      Delete patient
// @Description  Delete a patient record by ID. Associated records in junction tables (patient_symptoms, patient_disease) should be removed via ON DELETE CASCADE.
// @Tags         Patients
// @Accept       json
// @Produce      json
// @Param        patientID path      int true "Patient ID" Format(int32)
// @Success      204       {string}  string "No Content (Successful deletion)"
// @Failure      400       {object}  HTTPError "Invalid Patient ID format"
// @Failure      500       {object}  HTTPError "Internal server error"
// @Router       /patients/{patientID} [delete]
func (s *Server) handleDeletePatient() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		patientID, err := parseInt32Param(r, "patientID")
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid patient ID: "+err.Error())
			return
		}

		err = s.queries.DeletePatient(r.Context(), patientID)
		if err != nil {
			// Note: DELETE often doesn't error if the ID doesn't exist, but FK errors could occur if CASCADE isn't set up.
			log.Printf("Error deleting patient %d: %v", patientID, err)
			respondWithError(w, http.StatusInternalServerError, "Failed to delete patient")
			return
		}

		w.WriteHeader(http.StatusNoContent) // Standard response for successful DELETE
	}
}

// handleGetPatientDetails godoc
// @Summary      Get patient summary details
// @Description  Retrieve patient details along with aggregated lists of their general symptoms and distinct diseases recorded. Uses the GetPatientSummary query.
// @Tags         Patients
// @Accept       json
// @Produce      json
// @Param        patientID path      int true "Patient ID" Format(int32)
// @Success      200       {object}  PatientDetailsResponse "Successfully retrieved patient summary details"
// @Failure      400       {object}  HTTPError "Invalid Patient ID format"
// @Failure      404       {object}  HTTPError "Patient not found"
// @Failure      500       {object}  HTTPError "Internal server error"
// @Router       /patients/{patientID}/details [get]
func (s *Server) handleGetPatientDetails() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		patientID, err := parseInt32Param(r, "patientID")
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid patient ID: "+err.Error())
			return
		}

		// Use the GetPatientSummary query which returns aggregated data
		summary, err := s.queries.GetPatientSummary(r.Context(), patientID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows) {
				respondWithError(w, http.StatusNotFound, "Patient not found")
			} else {
				log.Printf("Error getting patient summary %d: %v", patientID, err)
				respondWithError(w, http.StatusInternalServerError, "Failed to retrieve patient summary")
			}
			return
		}

		// Parse the aggregated byte slices into string slices
		response := PatientDetailsResponse{
			PatientID:            summary.PatientID,
			Firstname:            summary.Firstname,
			Lastname:             summary.Lastname,
			Email:                summary.Email,
			GeneralSymptomsList:  parseAggregatedList(summary.GeneralSymptomsList),
			DistinctDiseasesList: parseAggregatedList(summary.DistinctDiseasesList),
		}

		respondWithJSON(w, http.StatusOK, response)
	}
}

