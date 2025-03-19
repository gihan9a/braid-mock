package config

import (
	"flag"
	"log"
	"net/url"
)

// TLSConfig holds TLS configuration options
type TLSConfig struct {
	Enabled      bool
	CertFile     string
	KeyFile      string
	GenerateCert bool
}

// CORSConfig holds CORS configuration options
type CORSConfig struct {
	Enabled          bool
	AllowOrigins     string
	AllowMethods     string
	AllowHeaders     string
	AllowCredentials bool
	MaxAge           int
}

// Config holds the application configuration
type Config struct {
	RootDir       string
	Port          int
	ProxyURL      *url.URL
	InsecureProxy bool
	TLS           TLSConfig
	CORS          CORSConfig
}

// ParseFlags parses command line flags and merges with config file
func ParseFlags() (*Config, error) {
	// Define flags
	configFlag := flag.String("config", "config.yml", "Path to configuration file")
	generateConfigFlag := flag.Bool("generate-config", false, "Generate a default configuration file")
	configFilePathFlag := flag.String("config-path", "config.yml", "Path where config file should be generated")

	// Simple flags for overriding config file
	dirFlag := flag.String("d", "", "Directory containing .braid mock files (overrides config)")
	portFlag := flag.Int("p", 0, "Port to listen on (overrides config)")

	// Parse flags
	flag.Parse()

	// Handle config file generation
	if *generateConfigFlag {
		log.Printf("Generating default configuration file at %s", *configFilePathFlag)
		if err := SaveDefaultConfig(*configFilePathFlag); err != nil {
			return nil, err
		}
		log.Printf("Configuration file generated successfully")
	}

	// Load configuration from file
	config, err := LoadConfig(*configFlag)
	if err != nil {
		log.Printf("Warning: Could not load config file: %v", err)
		log.Printf("Using default configuration")

		// If config file doesn't exist, use default config
		config, _ = LoadConfig("")
	}

	// Override with command line flags if provided
	if *dirFlag != "" {
		config.RootDir = *dirFlag
	}

	if *portFlag != 0 {
		config.Port = *portFlag
	}

	return config, nil
}
