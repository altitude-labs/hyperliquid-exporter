package config

import (
	"os"

	"github.com/joho/godotenv"
	"github.com/validaoxyz/hyperliquid-exporter/internal/logger"
)

type Config struct {
	HomeDir    string
	NodeHome   string
	BinaryHome string
	NodeBinary string
}

type Flags struct {
	NodeHome   string
	NodeBinary string
}

// loads env vars and returns a Config struct
func LoadConfig(flags *Flags) Config {
	// Load .env file first
	if err := godotenv.Load(); err != nil {
		logger.Debug("No .env file found, using environment variables and flags")
	}

	homeDir := os.Getenv("HOME")

	nodeHome := os.Getenv("NODE_HOME")
	if nodeHome == "" {
		nodeHome = homeDir + "/hl" //default fallback
	}

	binaryHome := os.Getenv("BINARY_HOME")
	if binaryHome == "" {
		binaryHome = homeDir
	}

	nodeBinary := os.Getenv("NODE_BINARY")
	if nodeBinary == "" {
		nodeBinary = binaryHome + "/hl-node"
	}

	config := Config{
		NodeHome:   nodeHome,
		NodeBinary: nodeBinary,
	}

	// override with flags if they're provided (non-empty)
	if flags != nil {
		if flags.NodeHome != "" {
			config.NodeHome = flags.NodeHome
		}
		if flags.NodeBinary != "" {
			config.NodeBinary = flags.NodeBinary
		}
	}

	return config
}
