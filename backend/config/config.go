package config

import (
	"fmt"

	"github.com/dukunuu/munkhjin-diplom/backend/common"
)

type Config struct {
	Port			string
	DB_Url		string
	Model_Url string
}

func Load() (*Config, error){
	port := common.GetString("PORT", ":8080");

	db := common.GetString("DB_URL", "");
	if db == "" {
		return nil, fmt.Errorf("DB_URL is required");
	}

	modelUrl := common.GetString("MODEL_URL", "http://flask_ml_service:5000/predict")

	return &Config{
		Port: port,
		DB_Url: db,
		Model_Url: modelUrl,
	}, nil
}
