package matching

import (
	"context"
	"math"
	"net"
	"reflect"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	commonv1 "order-fill/backend/proto/gen/go/orderfill/common/v1"
	matchingv1 "order-fill/backend/proto/gen/go/orderfill/matching/v1"
	"order-fill/backend/services/document-service/internal/domain/orderfill"
)

func TestGRPCMatchMapsRequestsAndResults(t *testing.T) {
	t.Parallel()

	server := &matchingServer{
		matchResponse: &matchingv1.MatchRowsResponse{Results: []*matchingv1.MatchResult{{
			BlankItemId:  "blank-1",
			SourceItemId: "source-1",
			Category:     commonv1.ReportCategory_REPORT_CATEGORY_NEEDS_DECISION,
			Reasons: &matchingv1.MatchReasons{
				Article: "alias", Name: "close", Volume: "match", Form: "conflict", Duplicates: "chosen_best", Source: "name",
			},
			Score:        0.75,
			CandidateIds: []string{"source-1", "source-2"},
		}}},
	}
	client := newTestClient(t, server)
	blank := []orderfill.MatchItem{{ID: "blank-1", Article: "A-1", Name: "Cream", Volume: "50 ml", Form: "cream", ChestnyZnak: true, Rounded: math.MaxInt32 + 1}}
	source := []orderfill.MatchItem{{ID: "source-1", Article: "MT-A-1", Name: "Cream", Rounded: math.MinInt32 - 1}}
	got, err := client.Match(t.Context(), blank, source, orderfill.MatchOptions{
		Mode: "smart", PrefixAliases: []string{"MT"}, PreserveHyphen: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if server.matchRequest.GetMatchingMode() != commonv1.MatchingMode_MATCHING_MODE_SMART ||
		!reflect.DeepEqual(server.matchRequest.GetPrefixAliases(), []string{"MT"}) || !server.matchRequest.GetPreserveHyphen() {
		t.Fatalf("request options = %+v", server.matchRequest)
	}
	if gotBlank := server.matchRequest.GetBlankItems()[0]; gotBlank.GetId() != "blank-1" || gotBlank.GetRounded() != math.MaxInt32 || !gotBlank.GetChestnyZnak() {
		t.Fatalf("blank item = %+v", gotBlank)
	}
	if gotSource := server.matchRequest.GetSourceItems()[0]; gotSource.GetId() != "source-1" || gotSource.GetRounded() != math.MinInt32 {
		t.Fatalf("source item = %+v", gotSource)
	}
	want := []orderfill.MatchResult{{
		BlankID: "blank-1", SourceID: "source-1", Category: orderfill.CategoryNeedsDecision,
		Score: 0.75, CandidateIDs: []string{"source-1", "source-2"},
		Reasons: orderfill.MatchReasons{Article: "alias", Name: "close", Volume: "match", Form: "conflict", Duplicates: "chosen_best", Source: "name"},
	}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Match() = %+v, want %+v", got, want)
	}
}

func TestGRPCMergeChzMapsRequestsAndResults(t *testing.T) {
	t.Parallel()

	server := &matchingServer{
		mergeResponse: &matchingv1.MergeChestnyZnakResponse{Merges: []*matchingv1.ChestnyZnakMerge{{
			TargetId: "base", CloneIds: []string{"chz"}, NeedsDecision: true,
		}}},
	}
	client := newTestClient(t, server)
	items := []orderfill.MatchItem{{ID: "chz", Article: "A-1", Rounded: math.MaxInt32 + 10}}
	got, err := client.MergeChz(t.Context(), items, orderfill.MatchOptions{
		Mode: "smart", PrefixAliases: []string{"MT"}, PreserveHyphen: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if server.mergeRequest.GetMatchingMode() != commonv1.MatchingMode_MATCHING_MODE_SMART ||
		!reflect.DeepEqual(server.mergeRequest.GetPrefixAliases(), []string{"MT"}) || !server.mergeRequest.GetPreserveHyphen() {
		t.Fatalf("request options = %+v", server.mergeRequest)
	}
	if gotItem := server.mergeRequest.GetItems()[0]; gotItem.GetId() != "chz" || gotItem.GetRounded() != math.MaxInt32 {
		t.Fatalf("item = %+v", gotItem)
	}
	want := []orderfill.ChzMerge{{TargetID: "base", CloneIDs: []string{"chz"}, NeedsDecision: true}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("MergeChz() = %+v, want %+v", got, want)
	}
}

func TestGRPCReturnsUpstreamErrors(t *testing.T) {
	t.Parallel()

	server := &matchingServer{matchErr: status.Error(codes.Unavailable, "matching unavailable"), mergeErr: status.Error(codes.Unavailable, "merge unavailable")}
	client := newTestClient(t, server)
	if _, err := client.Match(t.Context(), nil, nil, orderfill.MatchOptions{}); status.Code(err) != codes.Unavailable {
		t.Fatalf("Match() error = %v, want Unavailable", err)
	}
	if _, err := client.MergeChz(t.Context(), nil, orderfill.MatchOptions{}); status.Code(err) != codes.Unavailable {
		t.Fatalf("MergeChz() error = %v, want Unavailable", err)
	}
}

func TestInt32Clamp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   int
		want int32
	}{
		{name: "upper overflow", in: math.MaxInt32 + 1, want: math.MaxInt32},
		{name: "lower overflow", in: math.MinInt32 - 1, want: math.MinInt32},
		{name: "in range", in: 42, want: 42},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := int32Clamp(tt.in); got != tt.want {
				t.Fatalf("int32Clamp(%d) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}

type matchingServer struct {
	matchingv1.UnimplementedMatchingServiceServer
	matchRequest  *matchingv1.MatchRowsRequest
	mergeRequest  *matchingv1.MergeChestnyZnakRequest
	matchResponse *matchingv1.MatchRowsResponse
	mergeResponse *matchingv1.MergeChestnyZnakResponse
	matchErr      error
	mergeErr      error
}

func (s *matchingServer) MatchRows(_ context.Context, req *matchingv1.MatchRowsRequest) (*matchingv1.MatchRowsResponse, error) {
	s.matchRequest = req
	return s.matchResponse, s.matchErr
}

func (s *matchingServer) MergeChestnyZnak(_ context.Context, req *matchingv1.MergeChestnyZnakRequest) (*matchingv1.MergeChestnyZnakResponse, error) {
	s.mergeRequest = req
	return s.mergeResponse, s.mergeErr
}

func newTestClient(t *testing.T, implementation matchingv1.MatchingServiceServer) *GRPC {
	t.Helper()

	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	matchingv1.RegisterMatchingServiceServer(server, implementation)
	go func() {
		_ = server.Serve(listener)
	}()
	t.Cleanup(func() {
		server.Stop()
		_ = listener.Close()
	})

	conn, err := grpc.NewClient(
		"passthrough:///matching-test",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return &GRPC{client: matchingv1.NewMatchingServiceClient(conn)}
}

func TestMatchingModeMapsSmart(t *testing.T) {
	t.Parallel()
	if matchingMode("smart") != commonv1.MatchingMode_MATCHING_MODE_SMART {
		t.Fatal("smart must map to MATCHING_MODE_SMART")
	}
	if matchingMode("standard") != commonv1.MatchingMode_MATCHING_MODE_STANDARD {
		t.Fatal("standard must map to MATCHING_MODE_STANDARD")
	}
}

func TestCategoryNameIncludesNotInBlank(t *testing.T) {
	t.Parallel()
	if categoryName(commonv1.ReportCategory_REPORT_CATEGORY_NOT_IN_BLANK) != orderfill.CategoryNotInBlank {
		t.Fatalf("got %q", categoryName(commonv1.ReportCategory_REPORT_CATEGORY_NOT_IN_BLANK))
	}
}
