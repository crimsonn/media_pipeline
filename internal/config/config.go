package config

type Config struct {
	APIAddr        string
	APIPort        string
	Environment    string
	TranscoderAddr string
	DatabaseURL    string
}

func LoadConfig() *Config {
	transcoderAddr := GetEnvString("TRANSCODER_ADDR", ":50051")
	apiAddr := GetEnvString("API_ADDR", "localhost")
	apiPort := GetEnvString("API_PORT", "8080")
	environment := GetEnvString("ENVIRONMENT", "development")
	databaseURL := GetEnvString(
		"DATABASE_URL",
		"postgres://media:media@localhost:5432/media_pipeline?sslmode=disable",
	)
	return &Config{
		APIAddr:        apiAddr,
		APIPort:        apiPort,
		Environment:    environment,
		TranscoderAddr: transcoderAddr,
		DatabaseURL:    databaseURL,
	}
}
