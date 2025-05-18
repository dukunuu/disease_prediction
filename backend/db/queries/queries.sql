-- queries.sql -- Updated for Refined PostgreSQL Schema with sqlc

-- === Patient Queries ===

-- name: CreatePatient :one
INSERT INTO patient (
    firstname, lastname, register, age, gender, birthdate, address, phonenumber, email
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
)
RETURNING *;

-- name: GetPatientByID :one
SELECT * FROM patient
WHERE patient_id = $1 LIMIT 1;

-- name: GetPatientByEmail :one
SELECT * FROM patient
WHERE email = $1 LIMIT 1;

-- name: ListPatients :many
SELECT * FROM patient
ORDER BY lastname, firstname
LIMIT $1 OFFSET $2; -- For pagination

-- name: UpdatePatientDetails :one
-- Note: patient table in the schema provided doesn't have created_at/updated_at or triggers
UPDATE patient
SET
    firstname = $2,
    lastname = $3,
    register = $4,
    age = $5,
    gender = $6,
    birthdate = $7,
    address = $8,
    phonenumber = $9,
    email = $10
WHERE patient_id = $1
RETURNING *;

-- name: UpdatePatientAddress :one
-- Note: patient table in the schema provided doesn't have created_at/updated_at or triggers
UPDATE patient
SET
    address = $2
WHERE patient_id = $1
RETURNING *;

-- name: DeletePatient :exec
-- Note: ON DELETE CASCADE will handle related records in junction tables
DELETE FROM patient
WHERE patient_id = $1;


-- === Symptom Queries ===

-- name: CreateSymptom :one
INSERT INTO symptoms (
    symptom_name, symptom_description
) VALUES (
    $1, $2
)
RETURNING *;

-- name: GetSymptomByID :one
SELECT * FROM symptoms
WHERE symptom_id = $1 LIMIT 1;

-- name: ListSymptoms :many
SELECT * FROM symptoms
ORDER BY symptom_name;

-- name: UpdateSymptom :one
-- updated_at is handled by trigger_set_timestamp
UPDATE symptoms
SET
    symptom_name = $2,
    symptom_description = $3
WHERE symptom_id = $1
RETURNING *;

-- name: DeleteSymptom :exec
-- Note: ON DELETE CASCADE will handle related records in junction tables
DELETE FROM symptoms
WHERE symptom_id = $1;


-- === Disease Queries ===

-- name: CreateDisease :one
INSERT INTO disease (
    disease_name, disease_code, disease_description, disease_treatment
) VALUES (
    $1, $2, $3, $4 -- $4 should be valid JSON(B) text or compatible type
)
RETURNING *;

-- name: GetDiseaseByID :one
SELECT * FROM disease
WHERE disease_id = $1 LIMIT 1;

-- name: GetDiseaseByCode :one
SELECT * FROM disease
WHERE disease_code = $1 LIMIT 1;

-- name: ListDiseases :many
SELECT * FROM disease
ORDER BY disease_name;

-- name: UpdateDisease :one
-- updated_at is handled by trigger_set_timestamp
UPDATE disease
SET
    disease_name = $2,
    disease_code = $3,
    disease_description = $4,
    disease_treatment = $5 -- $5 should be valid JSON(B) text or compatible type
WHERE disease_id = $1
RETURNING *;

-- name: DeleteDisease :exec
-- Note: ON DELETE CASCADE will handle related records in junction tables
DELETE FROM disease
WHERE disease_id = $1;


-- === Patient Symptom Queries (General - Optional Table) ===
-- Use these if you need to record symptoms reported outside a specific diagnosis

-- name: RecordPatientSymptom :one
-- Records a general symptom for a patient, optionally with a date
INSERT INTO patient_symptoms (
    patient_id, symptom_id, reported_date
) VALUES (
    $1, $2, $3
)
RETURNING *;

-- name: RemovePatientSymptom :exec
-- Removes a specific general symptom record for a patient
DELETE FROM patient_symptoms
WHERE patient_id = $1 AND symptom_id = $2 AND reported_date = $3; -- More specific deletion

-- name: RemovePatientSymptomByID :exec
-- Removes a specific general symptom record by its ID
DELETE FROM patient_symptoms
WHERE id = $1;

-- name: ListGeneralSymptomsForPatient :many
-- Lists general symptoms recorded for a patient via patient_symptoms table
SELECT s.*, ps.reported_date
FROM symptoms s
JOIN patient_symptoms ps ON s.symptom_id = ps.symptom_id
WHERE ps.patient_id = $1
ORDER BY ps.reported_date DESC, s.symptom_name;

-- name: ListPatientsWithGeneralSymptom :many
-- Lists patients who reported a specific general symptom via patient_symptoms table
SELECT p.*
FROM patient p
JOIN patient_symptoms ps ON p.patient_id = ps.patient_id
WHERE ps.symptom_id = $1
ORDER BY p.lastname, p.firstname;


-- === Patient Disease Instance Queries ===

-- name: RecordPatientDiseaseInstance :one
-- Records a specific diagnosis instance for a patient
INSERT INTO patient_disease (
    patient_id, disease_id, diagnosis_date, notes
) VALUES (
    $1, $2, $3, $4
)
RETURNING *; -- Returns the newly created patient_disease record including patient_disease_id

-- name: GetPatientDiseaseInstanceByID :one
-- Gets a specific diagnosis instance by its unique ID
SELECT * FROM patient_disease
WHERE patient_disease_id = $1;

-- name: UpdatePatientDiseaseInstance :one
-- Updates details of a specific diagnosis instance
-- updated_at is handled by trigger_set_timestamp
UPDATE patient_disease
SET
    patient_id = $2,
    disease_id = $3,
    diagnosis_date = $4,
    notes = $5
WHERE patient_disease_id = $1
RETURNING *;

-- name: DeletePatientDiseaseInstance :exec
-- Deletes a specific diagnosis instance by its ID
-- Note: ON DELETE CASCADE handles related patient_disease_symptom records
DELETE FROM patient_disease
WHERE patient_disease_id = $1;

-- name: ListDiseaseInstancesForPatient :many
-- Lists all recorded disease instances for a specific patient
SELECT
    pd.patient_disease_id,
    pd.diagnosis_date,
    pd.notes,
    pd.created_at,
    pd.updated_at,
    d.disease_id,
    d.disease_name,
    d.disease_code
FROM patient_disease pd
JOIN disease d ON pd.disease_id = d.disease_id
WHERE pd.patient_id = $1
ORDER BY pd.diagnosis_date DESC, d.disease_name;

-- name: ListPatientsWithDiseaseInstance :many
-- Lists patients who have a recorded instance of a specific disease
SELECT p.*
FROM patient p
JOIN patient_disease pd ON p.patient_id = pd.patient_id
WHERE pd.disease_id = $1
ORDER BY p.lastname, p.firstname;


-- === Patient Disease Symptom Link Queries ===

-- name: LinkSymptomToPatientDisease :one
-- Links a symptom to a specific patient disease instance
INSERT INTO patient_disease_symptom (
    patient_disease_id, symptom_id
) VALUES (
    $1, $2
)
RETURNING *;

-- name: UnlinkSymptomFromPatientDisease :exec
-- Removes the link between a symptom and a specific patient disease instance
DELETE FROM patient_disease_symptom
WHERE patient_disease_id = $1 AND symptom_id = $2;

-- name: GetSymptomsForPatientDiseaseInstance :many
-- Gets symptoms linked to a specific disease instance by patient_disease_id
SELECT
    s.symptom_id,
    s.symptom_name,
    s.symptom_description
FROM symptoms s
JOIN patient_disease_symptom pds ON s.symptom_id = pds.symptom_id
WHERE pds.patient_disease_id = $1
ORDER BY s.symptom_name;


-- === Combined / Aggregate Queries ===

-- name: GetPatientDiseaseHistoryWithSymptoms :many
-- Get all disease instances and their linked symptoms for a specific patient
SELECT
    pt.patient_id,
    pt.firstname,
    pt.lastname,
    pd.patient_disease_id,
    d.disease_id,
    d.disease_name,
    pd.diagnosis_date,
    pd.notes AS disease_notes,
    s.symptom_id,
    s.symptom_name
FROM patient pt
JOIN patient_disease pd ON pt.patient_id = pd.patient_id
JOIN disease d ON pd.disease_id = d.disease_id
LEFT JOIN patient_disease_symptom pds ON pd.patient_disease_id = pds.patient_disease_id
LEFT JOIN symptoms s ON pds.symptom_id = s.symptom_id
WHERE pt.patient_id = $1
ORDER BY pd.diagnosis_date DESC, d.disease_name, s.symptom_name;


-- name: GetPatientSummary :one
-- Provides a summary overview for a single patient, aggregating general symptoms and distinct diseases.
-- Note: This uses the *optional* patient_symptoms table for the general symptom list.
-- Note: Disease list shows unique diseases ever recorded, not specific instances.
SELECT
    p.patient_id,
    p.firstname,
    p.lastname,
    p.email,
    -- Aggregate general symptoms from patient_symptoms
    (SELECT STRING_AGG(s_gen.symptom_name, ', ' ORDER BY s_gen.symptom_name)
     FROM symptoms s_gen
     JOIN patient_symptoms ps_gen ON s_gen.symptom_id = ps_gen.symptom_id
     WHERE ps_gen.patient_id = p.patient_id) AS general_symptoms_list,
    -- Aggregate distinct diseases from patient_disease
    (SELECT STRING_AGG(DISTINCT d_dis.disease_name, ', ' ORDER BY d_dis.disease_name)
     FROM disease d_dis
     JOIN patient_disease pd_dis ON d_dis.disease_id = pd_dis.disease_id
     WHERE pd_dis.patient_id = p.patient_id) AS distinct_diseases_list
FROM
    patient p
WHERE
    p.patient_id = $1
GROUP BY
    p.patient_id, p.firstname, p.lastname, p.email;


-- name: ListPatientSummaries :many
-- Provides a summary overview for all patients.
-- Similar aggregation logic as GetPatientSummary.
SELECT
    p.patient_id,
    p.firstname,
    p.lastname,
    p.email,
    (SELECT STRING_AGG(s_gen.symptom_name, ', ' ORDER BY s_gen.symptom_name)
     FROM symptoms s_gen
     JOIN patient_symptoms ps_gen ON s_gen.symptom_id = ps_gen.symptom_id
     WHERE ps_gen.patient_id = p.patient_id) AS general_symptoms_list,
    (SELECT STRING_AGG(DISTINCT d_dis.disease_name, ', ' ORDER BY d_dis.disease_name)
     FROM disease d_dis
     JOIN patient_disease pd_dis ON d_dis.disease_id = pd_dis.disease_id
     WHERE pd_dis.patient_id = p.patient_id) AS distinct_diseases_list
FROM
    patient p
GROUP BY
    p.patient_id, p.firstname, p.lastname, p.email
ORDER BY
    p.lastname, p.firstname;
