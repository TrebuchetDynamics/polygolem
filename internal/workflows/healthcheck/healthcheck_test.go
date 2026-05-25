package healthcheck

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestRunnerReportsHealthyDependencies(t *testing.T) {
	var calls []string
	runner := New(Config{
		Gamma: func(context.Context) error {
			calls = append(calls, "gamma")
			return nil
		},
		CLOB: func(context.Context) error {
			calls = append(calls, "clob")
			return nil
		},
	})

	got := runner.Run(context.Background())
	want := Result{"gamma": "ok", "clob": "ok"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("result=%+v, want %+v", got, want)
	}
	if !reflect.DeepEqual(calls, []string{"gamma", "clob"}) {
		t.Fatalf("calls=%v", calls)
	}
}

func TestRunnerReportsDependencyErrorsWithoutShortCircuit(t *testing.T) {
	var clobCalled bool
	runner := New(Config{
		Gamma: func(context.Context) error { return errors.New("gamma down") },
		CLOB: func(context.Context) error {
			clobCalled = true
			return errors.New("clob down")
		},
	})

	got := runner.Run(context.Background())
	want := Result{"gamma": "gamma down", "clob": "clob down"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("result=%+v, want %+v", got, want)
	}
	if !clobCalled {
		t.Fatal("CLOB check was skipped after Gamma failure")
	}
}
