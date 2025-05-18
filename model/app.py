import os
import joblib
import pandas as pd
from flask import Flask, request, jsonify
from scipy.special import softmax # For converting decision_function scores to probabilities
import numpy as np

# Updated filenames based on your new training script
MODEL_DIR = "model_files"
MODEL_SAVE_PATH = "disease_SVM.joblib"
FEATURES_SAVE_PATH = "svm_feature_names.joblib"
LABEL_ENCODER_SAVE_PATH = "label_encoder.joblib" # New artifact to load

model_file_path = os.path.join(MODEL_DIR, MODEL_SAVE_PATH)
features_file_path = os.path.join(MODEL_DIR, FEATURES_SAVE_PATH)
label_encoder_file_path = os.path.join(MODEL_DIR, LABEL_ENCODER_SAVE_PATH)

app = Flask(__name__)

# Load model, features, and label encoder
model = None
model_features = []
label_encoder = None
TOP_N_PREDICTIONS = 3 # How many top predictions to return

try:
    print(f"Loading model from {model_file_path}...")
    model = joblib.load(model_file_path)
    print("Model loaded successfully.")

    print(f"Loading features from {features_file_path}...")
    model_features = joblib.load(features_file_path)
    print(f"Model expects {len(model_features)} features.")

    print(f"Loading label encoder from {label_encoder_file_path}...")
    label_encoder = joblib.load(label_encoder_file_path)
    print("Label encoder loaded successfully.")

except FileNotFoundError as e:
    print("---------------------------------------------------------")
    print(f"ERROR: File not found. {e}")
    print(f"Looked for model at: {model_file_path}")
    print(f"Looked for features at: {features_file_path}")
    print(f"Looked for label encoder at: {label_encoder_file_path}")
    print("This likely means the training step failed or artifacts are missing.")
    print("---------------------------------------------------------")
except Exception as e:
    print(f"An error occurred during model/feature/encoder loading: {e}")


@app.route("/predict", methods=["POST"])
def predict():
    if model is None or not model_features or label_encoder is None:
        return jsonify({"error": "Model or related artifacts not loaded. Check server logs."}), 500

    try:
        data = request.get_json()
        if data is None:
            return jsonify({"error": "No JSON data received"}), 400

        # Expecting a structure like: {"symptoms": {"symptom1": 1, "symptom2": 1}}
        # or for batch: {"symptoms_list": [{"symptom1":1}, {"symptom2":1}]}
        # The notebook example uses a single dictionary of known symptoms.
        # Let's adapt to a single input for simplicity, matching the notebook's inference.
        # If batch is needed, the input structure and processing loop will need adjustment.

        known_symptoms = data.get("known_symptoms")
        if known_symptoms is None or not isinstance(known_symptoms, dict):
            return jsonify({
                "error": "Missing 'known_symptoms' key (must be a dictionary) in JSON data"
            }), 400

        # Build the input sample (list in fit-time column order)
        # This matches the logic in your notebook's prediction cell
        input_row = [known_symptoms.get(feat, 0) for feat in model_features]
        X_input = pd.DataFrame([input_row], columns=model_features)

        # Predict using SVM (with pseudo-probabilities from decision_function/softmax)
        scores = model.decision_function(X_input)

        # Ensure scores is 2D for softmax, even with one sample
        # For LinearSVC, decision_function for multi-class (ovr) returns (n_samples, n_classes)
        # If it's binary or only one sample for some reason makes it 1D:
        if scores.ndim == 1:
            # This case might happen if the model is binary and decision_function returns (n_samples,)
            # Or if it's multi-class but somehow squeezed.
            # For multi-class (n_classes > 2), decision_function should be (n_samples, n_classes)
            # If n_classes == 2, it's (n_samples,). We need to handle this for softmax.
            # A common way for binary with decision_function is to reshape for two "classes"
            # or apply sigmoid. Softmax expects multiple scores per sample.
            # Assuming multi-class as per typical disease prediction:
            if len(label_encoder.classes_) > 2:
                 scores = scores.reshape(1, -1) # Should already be (1, n_classes)
            elif len(label_encoder.classes_) == 2:
                # For binary, decision_function gives distance to hyperplane.
                # To use softmax, we might need to construct two "scores"
                # e.g., [-score, score] but this depends on interpretation.
                # Simpler: if binary, just predict and inverse_transform.
                # For now, let's assume the notebook's approach implies multi-class handling.
                # If your problem is strictly binary, this part might need adjustment.
                # If scores.shape is (1,) for a binary case, it means it's already (n_samples,).
                # Softmax on a single value isn't meaningful.
                # Let's stick to the notebook's direct application of softmax,
                # assuming `decision_function` output is appropriate for it.
                # The notebook's `scores.reshape(1, -1)` handles if it was (n_classes,) for a single sample.
                pass # Let softmax handle it, or adjust if errors occur for specific model outputs

        probs = softmax(scores, axis=1) # Probabilities for each class for each sample

        results = []
        # Since X_input is currently a single sample:
        current_probs = probs[0]
        top_n_indices = current_probs.argsort()[::-1][:TOP_N_PREDICTIONS]
        
        top_diseases = label_encoder.inverse_transform(top_n_indices)
        top_percentages = current_probs[top_n_indices] * 100

        predictions_output = []
        for disease, percent in zip(top_diseases, top_percentages):
            predictions_output.append({"disease": disease, "probability": f"{percent:.2f}%"})
        
        results.append(predictions_output) # For consistency if we extend to batch

        return jsonify({"predictions": results[0]}) # Return predictions for the single input

    except Exception as e:
        print(f"Error during prediction: {e}")
        import traceback
        traceback.print_exc()
        return jsonify({"error": f"An internal error occurred: {str(e)}"}), 500


@app.route("/", methods=["GET"])
def home():
    status = "Model, features, and label encoder loaded successfully."
    if model is None or not model_features or label_encoder is None:
        status = "Model or related artifacts failed to load. Check logs."
    return jsonify({"message": "Flask backend for Disease Prediction is running!", "model_status": status})


if __name__ == "__main__":
    # Make sure to install scipy: pip install scipy
    app.run(host="0.0.0.0", port=5000, debug=False)

