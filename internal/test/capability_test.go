//
// Copyright (C) 2019-2026 vdaas.org vald team <vald@vdaas.org>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// You may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package test

import (
	"context"
	"testing"
	"time"

	"github.com/vdaas/vald/internal/errors"
)

// TestCapabilities_T exercises every capability helper with X = *testing.T:
// Loop runs the body exactly once, Measured applies its timeout window to
// the single run and returns fn's error as-is via errors.Join, and all
// benchmark-only controls are safe no-ops.
func TestCapabilities_T(t *testing.T) {
	if IsBenchmark(t) {
		t.Error("IsBenchmark(*testing.T) must be false")
	}

	var runs int
	Loop(t, func() { runs++ })
	if runs != 1 {
		t.Errorf("Loop on *testing.T must run the body exactly once, ran %d times", runs)
	}

	if err := Measured(t.Context(), t, 100*time.Millisecond, func(ctx context.Context) error {
		if _, ok := ctx.Deadline(); !ok {
			return errors.New("measured context must carry the per-run timeout deadline")
		}
		return nil
	}); err != nil {
		t.Errorf("Measured returned unexpected error: %v", err)
	}

	wantErr := errors.New("measured failure")
	if err := Measured(t.Context(), t, 0, func(context.Context) error {
		return wantErr
	}); !errors.Is(err, wantErr) {
		t.Errorf("Measured must surface fn's error, got: %v", err)
	}

	// Benchmark-only controls must be no-ops on *testing.T.
	ReportMetric(t, 1.0, "noop")
	ReportAllocs(t)
	SetBytes(t, 1)
	ResetTimer(t)
	StartTimer(t)
	StopTimer(t)

	t.Run("parallel capability", func(tt *testing.T) {
		Parallel(tt) // *testing.T supports it; single subtest, so harmless.
	})

	// The constraint is plain testing.TB, so an interface-typed value must
	// also instantiate the helpers (X = testing.TB) with the capability
	// detection working off the dynamic type — the shape interface-typed
	// leaf helpers such as tests/v2/e2e/crud's logRecallAndQPS rely on.
	var tb testing.TB = t
	if IsBenchmark(tb) {
		t.Error("IsBenchmark must inspect the dynamic type behind testing.TB")
	}
	runs = 0
	Loop(tb, func() { runs++ })
	if runs != 1 {
		t.Errorf("Loop with X = testing.TB must run the body exactly once, ran %d times", runs)
	}
	ReportMetric(tb, 1.0, "noop")
}

// TestCapabilities_B exercises the helpers with X = *testing.B through
// testing.Benchmark. b.Loop consumes the benchmark's whole iteration
// budget, so Loop and Measured (which iterates via Loop) each get their own
// benchmark invocation — calling Loop twice in one invocation would make
// the second loop exit immediately, which is exactly the hazard Measured's
// single-Loop design avoids.
func TestCapabilities_B(t *testing.T) {
	var loops int
	res := testing.Benchmark(func(b *testing.B) {
		if !IsBenchmark(b) {
			b.Error("IsBenchmark(*testing.B) must be true")
		}
		ReportAllocs(b)
		ResetTimer(b)
		loops = 0
		Loop(b, func() { loops++ })
		StopTimer(b)
		SetBytes(b, 1)
		ReportMetric(b, float64(loops), "loops")
		StartTimer(b)
		Parallel(b) // no-op: *testing.B has no Parallel phase.
	})
	if loops < 1 {
		t.Errorf("Loop on *testing.B must run the body at least once, result: %s", res.String())
	}

	var iterations, misses int
	res = testing.Benchmark(func(b *testing.B) {
		iterations, misses = 0, 0
		if err := Measured(b.Context(), b, time.Second, func(ctx context.Context) error {
			iterations++
			if _, ok := ctx.Deadline(); !ok {
				misses++
			}
			return nil
		}); err != nil {
			b.Errorf("Measured returned unexpected error: %v", err)
		}
	})
	if iterations < 1 {
		t.Errorf("Measured on *testing.B must run fn at least once, result: %s", res.String())
	}
	if misses != 0 {
		t.Errorf("Measured must give every iteration a deadline-carrying context, %d/%d missed", misses, iterations)
	}
}
