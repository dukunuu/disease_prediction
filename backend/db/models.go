// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.29.0

package db

import (
	"github.com/jackc/pgx/v5/pgtype"
)

type Disease struct {
	DiseaseID          int32
	DiseaseName        string
	DiseaseCode        string
	DiseaseDescription pgtype.Text
	DiseaseTreatment   []byte
	CreatedAt          pgtype.Timestamp
	UpdatedAt          pgtype.Timestamp
}

type Patient struct {
	PatientID   int32
	Firstname   string
	Lastname    string
	Register    string
	Age         int32
	Gender      string
	Birthdate   pgtype.Date
	Address     pgtype.Text
	Phonenumber string
	Email       string
}

type PatientDisease struct {
	PatientDiseaseID int32
	PatientID        int32
	DiseaseID        int32
	DiagnosisDate    pgtype.Date
	Notes            pgtype.Text
	CreatedAt        pgtype.Timestamp
	UpdatedAt        pgtype.Timestamp
}

type PatientDiseaseSymptom struct {
	ID               int32
	PatientDiseaseID int32
	SymptomID        int32
	CreatedAt        pgtype.Timestamp
	UpdatedAt        pgtype.Timestamp
}

type PatientSymptom struct {
	ID           int32
	PatientID    int32
	SymptomID    int32
	ReportedDate pgtype.Date
	CreatedAt    pgtype.Timestamp
	UpdatedAt    pgtype.Timestamp
}

type Symptom struct {
	SymptomID          int32
	SymptomName        string
	SymptomDescription pgtype.Text
	CreatedAt          pgtype.Timestamp
	UpdatedAt          pgtype.Timestamp
}
