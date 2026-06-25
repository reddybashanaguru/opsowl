// license-gen is a CLI tool for OpsOwl staff to issue signed customer license
// files.  It is never shipped to customers.
//
// Usage:
//
//	go run ./cmd/license-gen \
//	  -key   /path/to/opsowl-license-private.pem \
//	  -org   "Acme Bank Ltd" \
//	  -seats 25 \
//	  -days  365 \
//	  -features insights,narrative,sanctions,gdpr,ai_act \
//	  -out   acme-bank.license
package main

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/reddybashanaguru/opsowl/internal/license"
)

func main() {
	keyPath  := flag.String("key",      "",    "path to OpsOwl RSA-2048 private key PEM (required)")
	org      := flag.String("org",      "",    "customer organisation name (required)")
	seats    := flag.Int("seats",       0,     "licensed seat count; 0 = unlimited")
	days     := flag.Int("days",        365,   "license validity in days")
	features := flag.String("features", "",    "comma-separated feature list (e.g. insights,narrative,sanctions)")
	outPath  := flag.String("out",      "",    "output file path (default: <org>.license)")
	flag.Parse()

	// ── Validate inputs ───────────────────────────────────────────────────────
	if *keyPath == "" {
		die("-key is required")
	}
	if *org == "" {
		die("-org is required")
	}
	if *days <= 0 {
		die("-days must be positive")
	}

	// ── Load private key ──────────────────────────────────────────────────────
	privKey, err := loadPrivateKey(*keyPath)
	if err != nil {
		die("load private key: %v", err)
	}

	// ── Build claims ──────────────────────────────────────────────────────────
	featureList := []string{}
	if *features != "" {
		for _, f := range strings.Split(*features, ",") {
			if trimmed := strings.TrimSpace(f); trimmed != "" {
				featureList = append(featureList, trimmed)
			}
		}
	}

	now     := time.Now().UTC()
	expiry  := now.Add(time.Duration(*days) * 24 * time.Hour)

	claims := license.Claims{
		OrgName:  *org,
		Seats:    *seats,
		Features: featureList,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "opsowl-licensing",
			Subject:   *org,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiry),
		},
	}

	// ── Sign ──────────────────────────────────────────────────────────────────
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, &claims)
	signed, err := token.SignedString(privKey)
	if err != nil {
		die("sign token: %v", err)
	}

	// ── Write output ──────────────────────────────────────────────────────────
	out := *outPath
	if out == "" {
		slug := strings.ToLower(strings.ReplaceAll(*org, " ", "-"))
		out  = slug + ".license"
	}
	if err := os.WriteFile(out, []byte(signed+"\n"), 0600); err != nil {
		die("write license file: %v", err)
	}

	fmt.Printf("License issued successfully\n")
	fmt.Printf("  File:     %s\n", out)
	fmt.Printf("  Org:      %s\n", *org)
	fmt.Printf("  Seats:    %d\n", *seats)
	fmt.Printf("  Features: %s\n", strings.Join(featureList, ", "))
	fmt.Printf("  Issued:   %s\n", now.Format("2006-01-02"))
	fmt.Printf("  Expires:  %s\n", expiry.Format("2006-01-02"))
	fmt.Println()
	fmt.Println("Deliver this file to the customer and instruct them to set:")
	fmt.Printf("  LICENSE_FILE=/path/to/%s\n", out)
}

func loadPrivateKey(path string) (*rsa.PrivateKey, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(raw)
	if block == nil {
		return nil, errors.New("no PEM block found in key file")
	}
	switch block.Type {
	case "RSA PRIVATE KEY":
		return x509.ParsePKCS1PrivateKey(block.Bytes)
	case "PRIVATE KEY":
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		rk, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("PKCS8 key is not RSA")
		}
		return rk, nil
	default:
		return nil, fmt.Errorf("unsupported PEM block type: %s", block.Type)
	}
}

func die(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "license-gen: "+format+"\n", args...)
	os.Exit(1)
}
