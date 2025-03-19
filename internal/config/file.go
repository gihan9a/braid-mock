package config

import (
	"fmt"
	"net/url"
	"os"

	"gopkg.in/yaml.v3"
)

// FileConfig represents the structure of the configuration file
type FileConfig struct {
	Server struct {
		Port    int    `yaml:"port"`
		RootDir string `yaml:"root_dir"`
	} `yaml:"server"`

	Proxy struct {
		URL            string `yaml:"url"`
		InsecureVerify bool   `yaml:"insecure_verify"`
	} `yaml:"proxy"`

	TLS struct {
		Enabled      bool   `yaml:"enabled"`
		CertFile     string `yaml:"cert_file"`
		KeyFile      string `yaml:"key_file"`
		GenerateCert bool   `yaml:"generate_cert"`
	} `yaml:"tls"`

	CORS struct {
		Enabled          bool   `yaml:"enabled"`
		AllowOrigins     string `yaml:"allow_origins"`
		AllowMethods     string `yaml:"allow_methods"`
		AllowHeaders     string `yaml:"allow_headers"`
		AllowCredentials bool   `yaml:"allow_credentials"`
		MaxAge           int    `yaml:"max_age"`
	} `yaml:"cors"`
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(filePath string) (*Config, error) {
	// Create default config
	config := &Config{
		RootDir:       ".",
		Port:          3000,
		InsecureProxy: false,
		TLS: TLSConfig{
			Enabled:      false,
			CertFile:     "cert/cert.pem",
			KeyFile:      "cert/key.pem",
			GenerateCert: false,
		},
		CORS: CORSConfig{
			Enabled:          false,
			AllowOrigins:     "*",
			AllowMethods:     "GET, POST, PUT, DELETE, OPTIONS, PATCH",
			AllowHeaders:     "Content-Type, Authorization, Subscribe, Version, Parents",
			AllowCredentials: false,
			MaxAge:           86400,
		},
	}

	// If no config file specified, return default config
	if filePath == "" {
		return config, nil
	}

	// Read config file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	// Parse YAML
	var fileConfig FileConfig
	if err := yaml.Unmarshal(data, &fileConfig); err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}

	// Update config with values from file
	if fileConfig.Server.Port != 0 {
		config.Port = fileConfig.Server.Port
	}
	if fileConfig.Server.RootDir != "" {
		config.RootDir = fileConfig.Server.RootDir
	}

	// Proxy settings
	if fileConfig.Proxy.URL != "" {
		proxyURL, err := url.Parse(fileConfig.Proxy.URL)
		if err != nil {
			return nil, fmt.Errorf("invalid proxy URL: %w", err)
		}
		config.ProxyURL = proxyURL
		config.InsecureProxy = fileConfig.Proxy.InsecureVerify
	}

	// TLS settings
	config.TLS.Enabled = fileConfig.TLS.Enabled
	if fileConfig.TLS.CertFile != "" {
		config.TLS.CertFile = fileConfig.TLS.CertFile
	}
	if fileConfig.TLS.KeyFile != "" {
		config.TLS.KeyFile = fileConfig.TLS.KeyFile
	}
	config.TLS.GenerateCert = fileConfig.TLS.GenerateCert

	// CORS settings
	config.CORS.Enabled = fileConfig.CORS.Enabled
	if fileConfig.CORS.AllowOrigins != "" {
		config.CORS.AllowOrigins = fileConfig.CORS.AllowOrigins
	}
	if fileConfig.CORS.AllowMethods != "" {
		config.CORS.AllowMethods = fileConfig.CORS.AllowMethods
	}
	if fileConfig.CORS.AllowHeaders != "" {
		config.CORS.AllowHeaders = fileConfig.CORS.AllowHeaders
	}
	config.CORS.AllowCredentials = fileConfig.CORS.AllowCredentials
	if fileConfig.CORS.MaxAge != 0 {
		config.CORS.MaxAge = fileConfig.CORS.MaxAge
	}

	return config, nil
}

// SaveDefaultConfig saves a default configuration file
func SaveDefaultConfig(filePath string) error {
	// Create default config structure
	var fileConfig FileConfig

	// Server settings
	fileConfig.Server.Port = 3000
	fileConfig.Server.RootDir = "."

	// Proxy settings
	fileConfig.Proxy.URL = ""
	fileConfig.Proxy.InsecureVerify = false

	// TLS settings
	fileConfig.TLS.Enabled = false
	fileConfig.TLS.CertFile = "cert/cert.pem"
	fileConfig.TLS.KeyFile = "cert/key.pem"
	fileConfig.TLS.GenerateCert = false

	// CORS settings
	fileConfig.CORS.Enabled = false
	fileConfig.CORS.AllowOrigins = "*"
	fileConfig.CORS.AllowMethods = "GET, POST, PUT, DELETE, OPTIONS, PATCH"
	fileConfig.CORS.AllowHeaders = "Content-Type, Authorization, Subscribe, Version, Parents"
	fileConfig.CORS.AllowCredentials = false
	fileConfig.CORS.MaxAge = 86400

	// Marshal to YAML
	data, err := yaml.Marshal(fileConfig)
	if err != nil {
		return fmt.Errorf("error creating default config: %w", err)
	}

	// Add helpful comments
	yamlWithComments := "# Braid Mock Server Configuration\n" +
		"# This file contains all settings for the Braid mock server\n\n" +
		string(data)

	// Write to file
	if err := os.WriteFile(filePath, []byte(yamlWithComments), 0644); err != nil {
		return fmt.Errorf("error writing config file: %w", err)
	}

	return nil
}
