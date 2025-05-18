package server

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"
)

type PredictRequest struct {
	KnownSymptoms any `json:"known_symptoms" binding:"required" example:"{\"feature1\": 10.5, \"feature2\": 2.3}"`
}

type PredictResponse struct {
	Predictions []string `json:"predictions" example:"[\"ClassA\", \"ClassB\"]"`
}

// predictHandler creates the HTTP handler function for proxying predictions.
// @Summary      Proxy Prediction Request
// @Description  Accepts feature data in JSON format, forwards it to the configured Flask ML service, and returns the prediction result. The 'features' field in the request body can be a single JSON object or an array of JSON objects.
// @Tags         predictions
// @Accept       json
// @Produce      json
// @Param        request body PredictRequest true "Prediction Request Features (single object or array of objects)"
// @Success      200  {object}  PredictResponse  "Successful prediction response (forwarded from Flask)"
// @Failure      400  {object}  HTTPError    "Bad Request - Invalid JSON format or missing 'features' key"
// @Failure      500  {object}  HTTPError    "Internal Server Error - Error during proxy processing or creating request"
// @Failure      502  {object}  HTTPError    "Bad Gateway - Failed to contact or get a valid response from the backend Flask service"
// @Router       /predict [post]
func (s *Server) predictHandler(flaskPredictURL string) (http.HandlerFunc) {
	client := &http.Client{
		Timeout: 30 * time.Second, // Timeout for the call to Flask
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var requestPayload PredictRequest
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("Error reading request body: %v", err)
			http.Error(w, "Error reading request body", http.StatusInternalServerError)
			return
		}
		defer r.Body.Close() // Close the original request body

		if err := json.Unmarshal(bodyBytes, &requestPayload); err != nil {
			log.Printf("Error decoding request JSON: %v. Body: %s", err, string(bodyBytes))
			respondWithError(w, http.StatusBadRequest, "Could not read body")
			return
		}

		if requestPayload.KnownSymptoms == nil {
			log.Printf("Missing 'known_symptoms' key in request. Body: %s", string(bodyBytes))
			respondWithError(w, http.StatusBadRequest, "Could not read known_symptoms in the request body.")
			return
		}

		flaskReq, err := http.NewRequest(http.MethodPost, flaskPredictURL, bytes.NewBuffer(bodyBytes))
		if err != nil {
			log.Printf("Error creating request to Flask: %v", err)
			respondWithError(w, http.StatusInternalServerError, "Could not create request.")
			return
		}
		flaskReq.Header.Set("Content-Type", "application/json")

		log.Printf("Forwarding request to %s", flaskPredictURL)
		flaskResp, err := client.Do(flaskReq)
		if err != nil {
			log.Printf("Error sending request to Flask: %v", err)
			respondWithError(w, http.StatusBadGateway, "Error sending request to Model service.")
			return
		}
		defer flaskResp.Body.Close()

		flaskRespBodyBytes, err := io.ReadAll(flaskResp.Body)
		if err != nil {
			log.Printf("Error reading Flask response body: %v", err)
			http.Error(w, "Error reading prediction service response", http.StatusInternalServerError)
			return
		}

		log.Printf("Received response from Flask - Status: %d, Body: %s", flaskResp.StatusCode, string(flaskRespBodyBytes))

		contentType := flaskResp.Header.Get("Content-Type")
		if contentType == "" {
			contentType = "application/json" // Assume JSON if Flask doesn't specify
		}
		w.Header().Set("Content-Type", contentType)

		w.WriteHeader(flaskResp.StatusCode)

		_, err = w.Write(flaskRespBodyBytes)
		if err != nil {
			log.Printf("Error writing response to client: %v", err)
		}
	}
}
