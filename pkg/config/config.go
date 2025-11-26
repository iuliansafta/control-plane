package config

import "github.com/iuliansafta/control-plane/pkg/utils"

type Config struct {
	GRPCPort     string
	HttpPort     string
	NomadAddr    string
	DBConnection string
	BootstrapKey string
	JWTSecret    string
}

func LoadConfig() *Config {
	return &Config{
		GRPCPort:     utils.GetEnv("GRPC_PORT", "50051"),
		HttpPort:     utils.GetEnv("HTTP_PORT", "5000"),
		NomadAddr:    utils.GetEnv("NOMAD_ADDR", "http://localhost:4646"),
		DBConnection: utils.GetEnv("DATABASE_URL", "postgres://postgres@localhost:5432/control-plane"),
		BootstrapKey: utils.GetEnv("BOOTSTRAP_KEY", ""),
		JWTSecret:    utils.GetEnv("JWT_SECRET", ""),
	}
}
