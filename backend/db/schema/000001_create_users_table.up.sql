-- PostgreSQL Schema

-- Function to update 'updated_at' timestamp
-- name: TriggerSetTimestampFunction
-- No specific sqlc command needed for function definition, but named for clarity
CREATE OR REPLACE FUNCTION trigger_set_timestamp()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Table: patient
-- name: PatientTable
CREATE TABLE patient (
    patient_id SERIAL PRIMARY KEY,
    firstname VARCHAR(255) NOT NULL,
    lastname VARCHAR(255) NOT NULL,
    register VARCHAR(100) NOT NULL,
    age INT NOT NULL,
    gender VARCHAR(255) NOT NULL,
    birthdate DATE NOT NULL,
    address VARCHAR(255),
    phonenumber VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE
);

-- Table: symptoms
-- name: SymptomsTable
CREATE TABLE symptoms (
    symptom_id SERIAL PRIMARY KEY,
    symptom_name VARCHAR(255) NOT NULL UNIQUE,
    symptom_description TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Trigger for symptoms
-- name: SetSymptomsTimestampTrigger
CREATE TRIGGER set_symptoms_timestamp
BEFORE UPDATE ON symptoms
FOR EACH ROW
EXECUTE FUNCTION trigger_set_timestamp();

-- Table: disease
-- name: DiseaseTable
CREATE TABLE disease (
    disease_id SERIAL PRIMARY KEY,
    disease_name VARCHAR(255) NOT NULL,
    disease_code VARCHAR(255) NOT NULL,
    disease_description TEXT,
    disease_treatment JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Trigger for disease
-- name: SetDiseaseTimestampTrigger
CREATE TRIGGER set_disease_timestamp
BEFORE UPDATE ON disease
FOR EACH ROW
EXECUTE FUNCTION trigger_set_timestamp();

-- Table: patient_disease (Represents a specific diagnosis/instance)
-- name: PatientDiseaseTable
CREATE TABLE patient_disease (
    patient_disease_id SERIAL PRIMARY KEY,
    patient_id INT NOT NULL,
    disease_id INT NOT NULL,
    diagnosis_date DATE,
    notes TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_pd_patient
        FOREIGN KEY (patient_id)
        REFERENCES patient(patient_id)
        ON DELETE CASCADE,
    CONSTRAINT fk_pd_disease
        FOREIGN KEY (disease_id)
        REFERENCES disease(disease_id)
        ON DELETE CASCADE,
    UNIQUE (patient_id, disease_id, diagnosis_date)
);

-- Trigger for patient_disease
-- name: SetPatientDiseaseTimestampTrigger
CREATE TRIGGER set_patient_disease_timestamp
BEFORE UPDATE ON patient_disease
FOR EACH ROW
EXECUTE FUNCTION trigger_set_timestamp();

-- Table: patient_disease_symptom (Links symptoms to a specific diagnosis)
-- name: PatientDiseaseSymptomTable
CREATE TABLE patient_disease_symptom (
    id SERIAL PRIMARY KEY,
    patient_disease_id INT NOT NULL,
    symptom_id INT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_pds_patient_disease
        FOREIGN KEY (patient_disease_id)
        REFERENCES patient_disease(patient_disease_id)
        ON DELETE CASCADE,
    CONSTRAINT fk_pds_symptom
        FOREIGN KEY (symptom_id)
        REFERENCES symptoms(symptom_id)
        ON DELETE CASCADE,
    UNIQUE (patient_disease_id, symptom_id)
);

-- Trigger for patient_disease_symptom
-- name: SetPatientDiseaseSymptomTimestampTrigger
CREATE TRIGGER set_patient_disease_symptom_timestamp
BEFORE UPDATE ON patient_disease_symptom
FOR EACH ROW
EXECUTE FUNCTION trigger_set_timestamp();


-- Table: patient_symptoms (Optional: General symptoms reported by patient)
-- name: PatientSymptomsTable
CREATE TABLE patient_symptoms (
    id SERIAL PRIMARY KEY,
    patient_id INT NOT NULL,
    symptom_id INT NOT NULL,
    reported_date DATE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_ps_patient
        FOREIGN KEY (patient_id)
        REFERENCES patient(patient_id)
        ON DELETE CASCADE,
    CONSTRAINT fk_ps_symptom
        FOREIGN KEY (symptom_id)
        REFERENCES symptoms(symptom_id)
        ON DELETE CASCADE,
    UNIQUE (patient_id, symptom_id, reported_date)
);

-- Trigger for patient_symptoms
-- name: SetPatientSymptomsTimestampTrigger
CREATE TRIGGER set_patient_symptoms_timestamp
BEFORE UPDATE ON patient_symptoms
FOR EACH ROW
EXECUTE FUNCTION trigger_set_timestamp();
