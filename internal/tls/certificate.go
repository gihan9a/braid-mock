package tls

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

// EnsureCertificate ensures a certificate exists, generating one if needed
func EnsureCertificate(certFile, keyFile string) error {
	// Check if certificate files already exist
	certExists := false
	keyExists := false

	if _, err := os.Stat(certFile); err == nil {
		certExists = true
	}

	if _, err := os.Stat(keyFile); err == nil {
		keyExists = true
	}

	// Generate only if both files don't exist
	if !certExists || !keyExists {
		return generateSelfSignedCert(certFile, keyFile)
	}

	log.Println("Using existing certificate files")
	return nil
}

// generateSelfSignedCert creates a self-signed certificate and key
func generateSelfSignedCert(certFile, keyFile string) error {
	log.Println("Generating self-signed certificate...")

	// Create certificate directory if it doesn't exist
	certDir := filepath.Dir(certFile)
	if _, err := os.Stat(certDir); os.IsNotExist(err) {
		if err := os.MkdirAll(certDir, 0755); err != nil {
			return fmt.Errorf("failed to create certificate directory: %w", err)
		}
	}

	// Generate private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate private key: %w", err)
	}

	// Prepare certificate template
	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour) // Valid for 1 year

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return fmt.Errorf("failed to generate serial number: %w", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Braid Mock Server"},
			CommonName:   "localhost",
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
		DNSNames:              []string{"localhost"},
	}

	// Create certificate
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return fmt.Errorf("failed to create certificate: %w", err)
	}

	// Write certificate to file
	certOut, err := os.Create(certFile)
	if err != nil {
		return fmt.Errorf("failed to open %s for writing: %w", certFile, err)
	}
	defer certOut.Close()

	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		return fmt.Errorf("failed to write certificate: %w", err)
	}

	// Write private key to file
	keyOut, err := os.OpenFile(keyFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to open %s for writing: %w", keyFile, err)
	}
	defer keyOut.Close()

	privBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	if err := pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: privBytes}); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	log.Printf("Generated self-signed certificate at %s", certFile)
	log.Printf("Generated private key at %s", keyFile)

	return nil
}
