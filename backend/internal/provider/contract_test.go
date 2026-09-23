package provider_test

import (
	"testing"

	"vps-billing/internal/provider"
	"vps-billing/internal/provider/mock"
	"vps-billing/internal/provider/providertest"
)

// RunProviderContractTests executes the standard provider contract test suite against any Provider implementation.
func RunProviderContractTests(t *testing.T, p provider.Provider) {
	providertest.RunProviderContractTests(t, p)
}

func TestMockProviderContract(t *testing.T) {
	mockProv := mock.NewMockProvider("test-mock")
	providertest.RunProviderContractTests(t, mockProv)
}
