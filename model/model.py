import pandas as pd
from sklearn.model_selection import StratifiedKFold, cross_val_score
from sklearn.preprocessing import LabelEncoder  # For LinearSVC
from sklearn.svm import LinearSVC  # New model from notebook
import joblib
import os

# Configuration
DATA_PATH = "model.csv"  # As per original script
TARGET_COLUMN = "Disease_name"  # As per original script

# File naming for saved artifacts (inspired by notebook)
MODEL_SAVE_PATH = "disease_SVM.joblib"
FEATURES_SAVE_PATH = "svm_feature_names.joblib"
LABEL_ENCODER_SAVE_PATH = "label_encoder.joblib"  # For the label encoder

MODEL_DIR = "model_files"  # As per original script

# Create directory if it doesn't exist
os.makedirs(MODEL_DIR, exist_ok=True)

# Construct full paths for saving artifacts
model_file_path = os.path.join(MODEL_DIR, MODEL_SAVE_PATH)
features_file_path = os.path.join(MODEL_DIR, FEATURES_SAVE_PATH)
label_encoder_file_path = os.path.join(MODEL_DIR, LABEL_ENCODER_SAVE_PATH)

# 1. Load Data
print(f"Loading data from {DATA_PATH}...")
try:
    df = pd.read_csv(DATA_PATH)
    print("Data loaded successfully.")
except FileNotFoundError:
    print(f"Error: Data file not found at {DATA_PATH}")
    print("Please ensure the path is correct and the file exists.")
    exit()

# 2. Prepare Data
print("Preparing data...")
if TARGET_COLUMN not in df.columns:
    print(f"Error: Target column '{TARGET_COLUMN}' not found in the dataframe.")
    print(f"Available columns: {df.columns.tolist()}")
    exit()

X = df.drop(TARGET_COLUMN, axis=1)
y_original = df[TARGET_COLUMN]
feature_names = list(
    X.columns
)  # Save original feature names, will be saved

print(f"Features ({len(feature_names)}): {feature_names}")
print(f"Target variable: {TARGET_COLUMN}")

# 3. Encode Target Variable (required for LinearSVC)
print("Encoding target variable...")
le = LabelEncoder()
y_encoded = le.fit_transform(y_original)
print(
    f"Target variable encoded. Number of classes: {len(le.classes_)}"
)  # Useful info

# 4. Initialize Model (LinearSVC from notebook)
print("Initializing LinearSVC model...")
# Hyperparameters from the notebook for LinearSVC
svm_params = {
    "C": 0.5,
    "class_weight": "balanced",
    "max_iter": 2000,
    "dual": True,  # As specified in the notebook. For LinearSVC, dual=True is
    # preferred when n_features > n_samples. It uses liblinear solver.
    # If scikit-learn version is >= 1.2, dual='auto' might be preferred
    # but we stick to the notebook's explicit setting.
}
# Add random_state for reproducibility, consistent with original script
svm_model = LinearSVC(**svm_params, random_state=42)

# 5. Cross-Validation (adapted from script, using new model and encoded y)
print("Performing 5-fold cross-validation...")
cv = StratifiedKFold(
    n_splits=5, shuffle=True, random_state=42
)  # Explicit KFold
# Cross-validation uses the numerically encoded target variable
cv_scores = cross_val_score(svm_model, X, y_encoded, cv=cv, scoring="accuracy")
print(f"Cross-Validation Accuracy Scores: {cv_scores}")
print(f"Mean CV Accuracy: {cv_scores.mean():.4f} (± {cv_scores.std():.4f})")

# 6. Train Final Model (on entire dataset, script's philosophy)
print("Training final model on the entire dataset...")
svm_model.fit(X, y_encoded)  # Train with all data and encoded target
print("Model training complete.")

# 7. Save Artifacts
# Save Model
print(f"Saving model to {model_file_path}...")
joblib.dump(svm_model, model_file_path)
print("Model saved.")

# Save Feature Names (in the exact training order)
print(f"Saving feature names to {features_file_path}...")
joblib.dump(
    feature_names, features_file_path
)  # feature_names derived from X.columns
print("Feature names saved.")

# Save Label Encoder (important for decoding predictions later)
print(f"Saving label encoder to {label_encoder_file_path}...")
joblib.dump(
    le, label_encoder_file_path
)  # 'le' is fitted on the full y_original
print("Label encoder saved.")

print("\nTraining and saving process finished.")

