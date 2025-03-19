package config

import (
	"flag"
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

// ParseFlags parses command line flags and returns a Config
func ParseFlags() (*Config, error) {
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

	// Define flags
	dirFlag := flag.String("d", config.RootDir, "Directory containing .braid mock files")
	portFlag := flag.Int("p", config.Port, "Port to listen on")
	proxyFlag := flag.String("proxy", "", "URL to proxy requests to when mock files aren't found")
	insecureProxyFlag := flag.Bool("insecure-proxy", config.InsecureProxy, "Skip SSL certificate verification when proxying requests")

	// TLS flags
	tlsFlag := flag.Bool("tls", config.TLS.Enabled, "Enable TLS (HTTPS)")
	certFlag := flag.String("cert", config.TLS.CertFile, "Path to TLS certificate file")
	keyFlag := flag.String("key", config.TLS.KeyFile, "Path to TLS private key file")
	genCertFlag := flag.Bool("gen-cert", config.TLS.GenerateCert, "Generate a self-signed certificate if none exists")

	// CORS flags
	corsFlag := flag.Bool("cors", config.CORS.Enabled, "Enable CORS support")
	corsOriginsFlag := flag.String("cors-origins", config.CORS.AllowOrigins, "Comma-separated list of allowed origins (e.g., '*' or 'https://example.com')")
	corsMethodsFlag := flag.String("cors-methods", config.CORS.AllowMethods, "Comma-separated list of allowed HTTP methods")
	corsHeadersFlag := flag.String("cors-headers", config.CORS.AllowHeaders, "Comma-separated list of allowed HTTP headers")
	corsCredentialsFlag := flag.Bool("cors-credentials", config.CORS.AllowCredentials, "Allow credentials (cookies, authorization headers, etc.)")
	corsMaxAgeFlag := flag.Int("cors-max-age", config.CORS.MaxAge, "Max age for CORS preflight requests in seconds")

	// Parse flags
	flag.Parse()

	// Update config
	config.RootDir = *dirFlag
	config.Port = *portFlag
	config.InsecureProxy = *insecureProxyFlag

	// TLS config
	config.TLS.Enabled = *tlsFlag
	config.TLS.CertFile = *certFlag
	config.TLS.KeyFile = *keyFlag
	config.TLS.GenerateCert = *genCertFlag

	// CORS config
	config.CORS.Enabled = *corsFlag
	config.CORS.AllowOrigins = *corsOriginsFlag
	config.CORS.AllowMethods = *corsMethodsFlag
	config.CORS.AllowHeaders = *corsHeadersFlag
	config.CORS.AllowCredentials = *corsCredentialsFlag
	config.CORS.MaxAge = *corsMaxAgeFlag

	// Parse proxy URL if specified
	if *proxyFlag != "" {
		proxyURL, err := url.Parse(*proxyFlag)
		if err != nil {
			return nil, err
		}
		config.ProxyURL = proxyURL
	}

	return config, nil
}
