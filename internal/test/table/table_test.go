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

package table

import (
	"context"
	"testing"

	"github.com/vdaas/vald/internal/errors"
)

// TestRun_T drives Run with X = *testing.T through the default check, a
// custom check with before/after hooks, an expected-error case, and a
// mismatch case whose error must be surfaced through Run's return value
// (the subtest itself does not fail; the caller decides).
func TestRun_T(t *testing.T) {
	wantErr := errors.New("expected failure")

	if err := Run(t.Context(), t, func(t *testing.T, in int) (int, error) {
		t.Helper()
		if in < 0 {
			return 0, wantErr
		}
		return in * 2, nil
	}, []CaseFor[*testing.T, int, int]{
		{Name: "default check matches", Args: 21, Want: Result[int]{Val: 42}},
		{Name: "expected error matches", Args: -1, Want: Result[int]{Err: wantErr}},
		{
			Name: "hooks and custom check",
			Args: 1,
			BeforeFunc: func(_ context.Context, t *testing.T, in int) int {
				t.Helper()
				return in + 2
			},
			CheckFunc: func(t *testing.T, _, got Result[int]) error {
				t.Helper()
				if got.Val != 6 {
					return errors.Errorf("before hook not applied, got %d", got.Val)
				}
				return got.Err
			},
			AfterFunc: func(_ context.Context, t *testing.T, _, _ int, err error) error {
				t.Helper()
				return err
			},
		},
	}...); err != nil {
		t.Errorf("Run returned unexpected error: %v", err)
	}

	if err := Run(t.Context(), t, func(t *testing.T, in int) (int, error) {
		t.Helper()
		return in, nil
	}, []CaseFor[*testing.T, int, int]{
		{Name: "mismatch is reported via return value", Args: 1, Want: Result[int]{Val: 2}},
	}...); err == nil {
		t.Error("Run must return the check error for a want/got mismatch")
	}
}

// TestRun_B drives Run with X = *testing.B through testing.Benchmark,
// proving the same table-driven runner handles benchmarks: each case
// becomes a b.Run sub-benchmark.
func TestRun_B(t *testing.T) {
	var ran bool
	res := testing.Benchmark(func(b *testing.B) {
		b.Helper()
		if err := Run(b.Context(), b, func(b *testing.B, in int) (int, error) {
			b.Helper()
			s := 0
			for b.Loop() {
				s += in
			}
			ran = true
			return in * 2, nil
		}, []CaseFor[*testing.B, int, int]{
			{Name: "sum", Args: 21, Want: Result[int]{Val: 42}},
		}...); err != nil {
			b.Errorf("Run returned unexpected error: %v", err)
		}
	})
	if !ran {
		t.Errorf("benchmark body did not run, result: %s", res.String())
	}
}
