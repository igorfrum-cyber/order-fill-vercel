package grpcapi

import (
	"testing"

	commonv1 "order-fill/backend/proto/gen/go/orderfill/common/v1"
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

func TestProtoCompanyMapsChristinaProffMode(t *testing.T) {
	t.Parallel()
	company := protoCompany(domain.Company{ID: "co-1", ChristinaProffMode: domain.ChristinaProffModeCompare})
	if company.GetChristinaProffMode() != commonv1.ChristinaProffMode_CHRISTINA_PROFF_MODE_COMPARE {
		t.Fatalf("got %v", company.GetChristinaProffMode())
	}
	if domainChristinaProffMode(commonv1.ChristinaProffMode_CHRISTINA_PROFF_MODE_UNSPECIFIED) != domain.ChristinaProffModeStandard {
		t.Fatal("unspecified must be standard")
	}
}
