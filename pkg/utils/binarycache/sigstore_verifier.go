package binarycache

import (
	"crypto"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/sigstore/sigstore-go/pkg/bundle"
	"github.com/sigstore/sigstore-go/pkg/root"
	"github.com/sigstore/sigstore-go/pkg/verify"
)

// SigstoreVerifier verifies binaries using sigstore-go library
type SigstoreVerifier struct {
	trustedRoot               *root.TrustedRoot
	verifier                  *verify.Verifier
	certificateIdentityRegexp string
	certificateOIDCIssuer     string
}

// NewSigstoreVerifier creates a new sigstore verifier using the public good instance
func NewSigstoreVerifier(cfg *Config) (*SigstoreVerifier, error) {
	// Use Sigstore Public Good Instance trusted root
	// This fetches the trusted root from TUF (The Update Framework)
	trustedRoot, err := root.FetchTrustedRoot()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch trusted root: %w", err)
	}

	verifierConfig := []verify.VerifierOption{
		verify.WithSignedCertificateTimestamps(1), // Require SCT
		verify.WithTransparencyLog(1),             // Require Rekor entry
		verify.WithObserverTimestamps(1),          // Require timestamp
	}

	verifier, err := verify.NewVerifier(trustedRoot, verifierConfig...)
	if err != nil {
		return nil, fmt.Errorf("failed to create verifier: %w", err)
	}

	return &SigstoreVerifier{
		trustedRoot:               trustedRoot,
		verifier:                  verifier,
		certificateIdentityRegexp: cfg.GetSigstoreIdentityRegexp(),
		certificateOIDCIssuer:     cfg.GetSigstoreOIDCIssuer(),
	}, nil
}

// VerifyBlob verifies a blob (file) using a sigstore bundle
func (sv *SigstoreVerifier) VerifyBlob(blobPath, bundlePath string) error {
	// Load the bundle
	b, err := bundle.LoadJSONFromPath(bundlePath)
	if err != nil {
		return fmt.Errorf("failed to load bundle: %w", err)
	}

	artifactBytes, err := os.ReadFile(blobPath)
	if err != nil {
		return fmt.Errorf("failed to read artifact: %w", err)
	}

	digest := sha256Digest(artifactBytes)
	artifactDigestBytes, err := hex.DecodeString(digest)
	if err != nil {
		return fmt.Errorf("failed to decode digest: %w", err)
	}

	certIdentity, err := verify.NewShortCertificateIdentity(
		sv.certificateOIDCIssuer,
		"",                           // issuer regex (empty = exact match)
		"",                           // SAN (empty = use regex)
		sv.certificateIdentityRegexp, // SAN regex
	)
	if err != nil {
		return fmt.Errorf("failed to create certificate identity: %w", err)
	}

	policy := verify.NewPolicy(
		verify.WithArtifactDigest("sha256", artifactDigestBytes),
		verify.WithCertificateIdentity(certIdentity),
	)

	_, err = sv.verifier.Verify(b, policy)
	if err != nil {
		return fmt.Errorf("verification failed: %w", err)
	}

	return nil
}

// sha256Digest calculates the SHA256 digest of data
func sha256Digest(data []byte) string {
	hash := crypto.SHA256.New()
	hash.Write(data)
	return hex.EncodeToString(hash.Sum(nil))
}
