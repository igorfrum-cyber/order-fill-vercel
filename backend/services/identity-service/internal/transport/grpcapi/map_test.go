package grpcapi

import (
	"testing"

	"order-fill/backend/services/identity-service/internal/domain"
)

func TestProtoPublicCompanyOmitsOrderProfile(t *testing.T) {
	t.Parallel()
	company := protoPublicCompany(domain.Company{ID: "co-1", Name: "Acme", OrderProfile: domain.OrderProfile{LegalName: "ООО Секрет"}})
	if company.GetOrderProfile() != nil {
		t.Fatalf("public company exposed order profile: %+v", company.GetOrderProfile())
	}
	if company.GetName() != "Acme" {
		t.Fatalf("name=%q", company.GetName())
	}
}
