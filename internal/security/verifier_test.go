package security

import (
	"context"
	"testing"
	"time"
)

func TestNewVerifier(t *testing.T) {
	verifier := NewVerifier(true, []string{"key1", "key2"})

	if verifier == nil {
		t.Fatal("expected non-nil verifier")
	}

	if !verifier.requireSignature {
		t.Error("expected requireSignature to be true")
	}

	if len(verifier.trustedKeys) != 2 {
		t.Errorf("expected 2 trusted keys, got %d", len(verifier.trustedKeys))
	}
}

func TestVerifyImage(t *testing.T) {
	tests := []struct {
		name             string
		image            string
		requireSignature bool
		wantVerified     bool
		wantError        bool
	}{
		{
			name:             "signed image from gcr.io",
			image:            "gcr.io/distroless/static:latest",
			requireSignature: true,
			wantVerified:     true,
			wantError:        false,
		},
		{
			name:             "signed image from ghcr.io",
			image:            "ghcr.io/sigstore/cosign:latest",
			requireSignature: true,
			wantVerified:     true,
			wantError:        false,
		},
		{
			name:             "unsigned image with signature required",
			image:            "nginx:latest",
			requireSignature: true,
			wantVerified:     false,
			wantError:        true,
		},
		{
			name:             "unsigned image without signature required",
			image:            "nginx:latest",
			requireSignature: false,
			wantVerified:     true,
			wantError:        false,
		},
		{
			name:             "sigstore official image",
			image:            "docker.io/sigstore/cosign:latest",
			requireSignature: true,
			wantVerified:     true,
			wantError:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verifier := NewVerifier(tt.requireSignature, []string{})
			ctx := context.Background()

			result, err := verifier.VerifyImage(ctx, tt.image)

			if (err != nil) != tt.wantError {
				t.Errorf("VerifyImage() error = %v, wantError %v", err, tt.wantError)
			}

			if result.Verified != tt.wantVerified {
				t.Errorf("VerifyImage() verified = %v, want %v", result.Verified, tt.wantVerified)
			}

			if result.Image != tt.image {
				t.Errorf("VerifyImage() image = %v, want %v", result.Image, tt.image)
			}
		})
	}
}

func TestVerifyImageWithTimeout(t *testing.T) {
	verifier := NewVerifier(false, []string{})

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	result, err := verifier.VerifyImage(ctx, "nginx:latest")
	if err != nil {
		t.Errorf("VerifyImage() unexpected error: %v", err)
	}

	if !result.Verified {
		t.Error("expected image to be verified when signature not required")
	}
}

func TestVerifyBatch(t *testing.T) {
	verifier := NewVerifier(false, []string{})
	ctx := context.Background()

	images := []string{
		"nginx:latest",
		"redis:7",
		"postgres:15",
	}

	results, err := verifier.VerifyBatch(ctx, images)
	if err != nil {
		t.Errorf("VerifyBatch() error = %v", err)
	}

	if len(results) != len(images) {
		t.Errorf("expected %d results, got %d", len(images), len(results))
	}

	for i, result := range results {
		if result.Image != images[i] {
			t.Errorf("result[%d] image = %v, want %v", i, result.Image, images[i])
		}
	}
}

func TestVerifyBatchWithSignatureRequired(t *testing.T) {
	verifier := NewVerifier(true, []string{})
	ctx := context.Background()

	images := []string{
		"gcr.io/distroless/static:latest", // Should pass
		"nginx:latest",                    // Should fail
	}

	results, err := verifier.VerifyBatch(ctx, images)

	// Should have error because one image failed
	if err == nil {
		t.Error("expected error for batch with failed verification")
	}

	if len(results) != len(images) {
		t.Errorf("expected %d results, got %d", len(images), len(results))
	}

	// First image should be verified
	if !results[0].Verified {
		t.Error("expected first image to be verified")
	}

	// Second image should not be verified
	if results[1].Verified {
		t.Error("expected second image to not be verified")
	}
}

func TestHasKnownSignature(t *testing.T) {
	verifier := NewVerifier(true, []string{})

	tests := []struct {
		name  string
		image string
		want  bool
	}{
		{"gcr.io image", "gcr.io/distroless/static:latest", true},
		{"ghcr.io image", "ghcr.io/owner/repo:tag", true},
		{"sigstore image", "docker.io/sigstore/cosign:latest", true},
		{"regular docker hub", "nginx:latest", false},
		{"regular docker hub with prefix", "docker.io/library/nginx:latest", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := verifier.hasKnownSignature(tt.image)
			if got != tt.want {
				t.Errorf("hasKnownSignature(%s) = %v, want %v", tt.image, got, tt.want)
			}
		})
	}
}

func TestPolicyCheckPolicy(t *testing.T) {
	tests := []struct {
		name    string
		policy  *Policy
		result  *VerificationResult
		wantErr bool
	}{
		{
			name: "verified image passes policy",
			policy: &Policy{
				RequireSignature: true,
				MinSignatures:    1,
			},
			result: &VerificationResult{
				Image:    "test:latest",
				Verified: true,
				Signatures: []SignatureInfo{
					{Issuer: "test-issuer", Subject: "test@example.com"},
				},
			},
			wantErr: false,
		},
		{
			name: "unverified image fails policy",
			policy: &Policy{
				RequireSignature: true,
			},
			result: &VerificationResult{
				Image:    "test:latest",
				Verified: false,
			},
			wantErr: true,
		},
		{
			name: "insufficient signatures",
			policy: &Policy{
				RequireSignature: true,
				MinSignatures:    2,
			},
			result: &VerificationResult{
				Image:    "test:latest",
				Verified: true,
				Signatures: []SignatureInfo{
					{Issuer: "test-issuer"},
				},
			},
			wantErr: true,
		},
		{
			name: "trusted issuer check passes",
			policy: &Policy{
				RequireSignature: true,
				TrustedIssuers:   []string{"trusted-issuer"},
				MinSignatures:    1,
			},
			result: &VerificationResult{
				Image:    "test:latest",
				Verified: true,
				Signatures: []SignatureInfo{
					{Issuer: "trusted-issuer", Subject: "test@example.com"},
				},
			},
			wantErr: false,
		},
		{
			name: "trusted issuer check fails",
			policy: &Policy{
				RequireSignature: true,
				TrustedIssuers:   []string{"trusted-issuer"},
				MinSignatures:    1,
			},
			result: &VerificationResult{
				Image:    "test:latest",
				Verified: true,
				Signatures: []SignatureInfo{
					{Issuer: "untrusted-issuer", Subject: "test@example.com"},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.policy.CheckPolicy(tt.result)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckPolicy() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestVerificationResultStructure(t *testing.T) {
	result := &VerificationResult{
		Image:    "test:latest",
		Verified: true,
		Signatures: []SignatureInfo{
			{
				Issuer:    "issuer1",
				Subject:   "subject1",
				Timestamp: "2026-04-06T00:00:00Z",
			},
		},
	}

	if result.Image != "test:latest" {
		t.Errorf("expected image test:latest, got %s", result.Image)
	}

	if !result.Verified {
		t.Error("expected verified to be true")
	}

	if len(result.Signatures) != 1 {
		t.Errorf("expected 1 signature, got %d", len(result.Signatures))
	}

	if result.Signatures[0].Issuer != "issuer1" {
		t.Errorf("expected issuer issuer1, got %s", result.Signatures[0].Issuer)
	}
}

func BenchmarkVerifyImage(b *testing.B) {
	verifier := NewVerifier(false, []string{})
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		verifier.VerifyImage(ctx, "nginx:latest")
	}
}

func BenchmarkVerifyBatch(b *testing.B) {
	verifier := NewVerifier(false, []string{})
	ctx := context.Background()
	images := []string{"nginx:latest", "redis:7", "postgres:15"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		verifier.VerifyBatch(ctx, images)
	}
}
