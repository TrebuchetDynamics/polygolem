package eventreads

import (
	"context"
	"reflect"
	"testing"

	"github.com/TrebuchetDynamics/polygolem/internal/polytypes"
)

type fakeReader struct {
	limit int
}

func (f *fakeReader) Events(_ context.Context, params *polytypes.GetEventsParams) ([]polytypes.Event, error) {
	f.limit = params.Limit
	return []polytypes.Event{{ID: "evt-1", Slug: "event-one", Title: "Event One"}}, nil
}

func TestRunnerListsEventsWithRequestedLimit(t *testing.T) {
	reader := &fakeReader{}
	runner := New(reader)

	got, err := runner.Run(context.Background(), Request{Limit: 3})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	want := []polytypes.Event{{ID: "evt-1", Slug: "event-one", Title: "Event One"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("events=%+v, want %+v", got, want)
	}
	if reader.limit != 3 {
		t.Fatalf("limit=%d, want 3", reader.limit)
	}
}
