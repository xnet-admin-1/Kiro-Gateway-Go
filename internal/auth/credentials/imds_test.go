package credentials

import (
	"context"
	"testing"
)

func TestIMDSProvider_Retrieve(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "IMDS not available (expected in test environment)",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewIMDSProvider()
			_, err := provider.Retrieve(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("IMDSProvider.Retrieve() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// In a real EC2 environment, we would test successful retrieval
			// but in our test environment, IMDS is not available
		})
	}
}

func TestIMDSProvider_IsExpired(t *testing.T) {
	provider := NewIMDSProvider()
	if provider.IsExpired() {
		t.Error("IMDSProvider.IsExpired() should always return false")
	}
}
