package server

import (
	"log"
	"net/http"

	"github.com/dukunuu/munkhjin-diplom/backend/db" // Your sqlc package
	_ "github.com/dukunuu/munkhjin-diplom/backend/docs" // Adjust path to your generated docs
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

type Server struct {
	pool    *pgxpool.Pool
	queries *db.Queries
	router  *chi.Mux
	// Add other fields like modelUrl if needed
	modelUrl string
}

// Assume Init function initializes pool, queries, router, modelUrl
func Init(pool *pgxpool.Pool, modelUrl string) *Server {
	queries := db.New(pool)
	router := chi.NewRouter()

	server := &Server{
		pool:     pool,
		router:   router,
		queries:  queries,
		modelUrl: modelUrl, // Store modelUrl if predictHandler needs it
	}

	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger) // Log requests
	router.Use(middleware.Recoverer) // Recover from panics

	server.setupRoutes() // Call setupRoutes internally
	return server
}

func (s *Server) setupRoutes() { // Removed modelUrl param, use s.modelUrl if needed
	s.router.Get("/swagger/*", httpSwagger.WrapHandler)

	// Simple welcome endpoint
	s.router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Welcome to the Patient API! View docs at /swagger/index.html"))
	})

	// --- Patient Base Routes ---
	s.router.Route("/patients", func(r chi.Router) {
		r.Get("/", s.handleListPatients())          // GET /patients?limit=10&offset=0
		r.Post("/", s.handleCreatePatient())         // POST /patients
		r.Get("/{patientID}", s.handleGetPatientByID()) // GET /patients/123
		r.Put("/{patientID}", s.handleUpdatePatientDetails()) // PUT /patients/123
		r.Delete("/{patientID}", s.handleDeletePatient())   // DELETE /patients/123

		r.Get("/{patientID}/details", s.handleGetPatientDetails())

		// --- General Symptoms Reported by Patient (patient_symptoms table) ---
		r.Route("/{patientID}/general-symptoms", func(gsr chi.Router) {
			gsr.Get("/", s.handleListGeneralSymptomsForPatient()) // GET /patients/123/general-symptoms
			gsr.Post("/", s.handleRecordPatientSymptom())         // POST /patients/123/general-symptoms
			// Deletion is handled by /patient-symptoms/{id}
		})

		// --- Disease Instances Recorded for Patient (patient_disease table) ---
		r.Route("/{patientID}/disease-instances", func(dir chi.Router) {
			dir.Get("/", s.handleListDiseaseInstancesForPatient()) // GET /patients/123/disease-instances
			dir.Post("/", s.handleRecordPatientDiseaseInstance())  // POST /patients/123/disease-instances
		})
	})

	// --- Direct Management of General Symptom Records ---
	s.router.Route("/patient-symptoms", func(r chi.Router) {
		r.Delete("/{id}", s.handleDeletePatientSymptom()) // DELETE /patient-symptoms/5
	})

	// --- Direct Management of Disease Instances & Their Linked Symptoms ---
	s.router.Route("/disease-instances", func(r chi.Router) {
		r.Delete("/{instanceID}", s.handleDeletePatientDiseaseInstance()) // DELETE /disease-instances/10

		r.Route("/{instanceID}/symptoms", func(disr chi.Router) {
			disr.Get("/", s.handleGetSymptomsForDiseaseInstance()) // GET /disease-instances/10/symptoms
			disr.Post("/", s.handleLinkSymptomToDiseaseInstance()) // POST /disease-instances/10/symptoms
			disr.Delete("/{symptomID}", s.handleUnlinkSymptomFromDiseaseInstance()) // DELETE /disease-instances/10/symptoms/456
		})
	})

	s.router.Route("/symptoms", func(r chi.Router) {
		r.Get("/", s.handleListSymptoms())        // GET /symptoms
		r.Post("/", s.handleCreateSymptom())       // POST /symptoms
		r.Get("/{symptomID}", s.handleGetSymptomByID()) // GET /symptoms/456
		r.Put("/{symptomID}", s.handleUpdateSymptom())   // PUT /symptoms/456
		r.Delete("/{symptomID}", s.handleDeleteSymptom()) // DELETE /symptoms/456
	})

	s.router.Route("/diseases", func(r chi.Router) {
		r.Get("/", s.handleListDiseases())        // GET /diseases
		r.Post("/", s.handleCreateDisease())       // POST /diseases
		r.Get("/{diseaseID}", s.handleGetDiseaseByID()) // GET /diseases/789
		r.Put("/{diseaseID}", s.handleUpdateDisease())   // PUT /diseases/789
		r.Delete("/{diseaseID}", s.handleDeleteDisease()) // DELETE /diseases/789
	})

	s.router.Post("/predict", s.predictHandler(s.modelUrl))

	s.router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
}

func (s *Server) Start(addr string) error {
	log.Printf("Server listening on %s", addr)
	log.Printf("API docs available at http://%s/swagger/index.html", addr) // Log swagger URL
	return http.ListenAndServe(addr, s.router)
}
