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

// This file is the capability layer that completes the Runner[X]
// unification: Runner expresses the shared testing.TB + Run surface, but
// the benchmark-only controls (b.Loop, b.ReportMetric, timer control, ...)
// and the test-only ones (t.Parallel) have no common interface, so generic
// code would otherwise scatter `any(t).(*testing.B)` assertions at every
// use site. Each helper below performs exactly one capability check against
// the narrow interface it needs — not against the concrete *testing.B /
// *testing.T types, so wrappers embedding them keep working — and degrades
// to a documented fallback when the capability is absent. The constraint is
// plain testing.TB (looser than Runner[X]) so both Runner-generic
// orchestration code and interface-typed leaf helpers can call them.

// unwrap follows Unwrap() testing.TB links (the convention Node
// implements, mirroring errors.Unwrap) so capability detection always
// inspects the concrete testing entry, no matter how many wrapper layers
// sit above it. The depth bound keeps a misbehaving self-returning Unwrap
// from spinning (wrappers holding closures are not comparable, so a
// same-value check is not an option).
func unwrap(t testing.TB) testing.TB {
	for range 8 {
		u, ok := t.(interface{ Unwrap() testing.TB })
		if !ok {
			return t
		}
		inner := u.Unwrap()
		if inner == nil {
			return t
		}
		t = inner
	}
	return t
}

// IsBenchmark reports whether t is driven by the benchmark harness,
// detected through the Loop capability rather than the concrete type.
func IsBenchmark[X testing.TB](t X) bool {
	_, ok := unwrap(t).(interface{ Loop() bool })
	return ok
}

// Loop executes body once per measured iteration when t exposes the
// benchmark Loop capability (b.Loop, which also confines the benchmark
// timer to the loop) and exactly once on any other testing.TB value. It is
// the unified "measured region" iteration primitive.
func Loop[X testing.TB](t X, body func()) {
	t.Helper()
	if l, ok := unwrap(t).(interface{ Loop() bool }); ok {
		for l.Loop() {
			body()
		}
		return
	}
	body()
}

// Measured runs fn as t's measured unit: each Loop iteration executes fn
// once with its own fresh timeout window when timeout > 0 (a single shared
// window would start expiring before the first iteration and starve later
// ones), and per-iteration errors are joined so an early failure is not
// masked by later successes. On non-benchmark Runners fn runs exactly once
// under the same per-run window semantics.
func Measured[X testing.TB](
	ctx context.Context, t X, timeout time.Duration, fn func(context.Context) error,
) (err error) {
	t.Helper()
	Loop(t, func() {
		ierr := func() error {
			ictx := ctx
			if timeout > 0 {
				var cancel context.CancelFunc
				ictx, cancel = context.WithTimeout(ctx, timeout)
				defer cancel()
			}
			return fn(ictx)
		}()
		if ierr != nil {
			err = errors.Join(err, ierr)
		}
	})
	return err
}

// ReportMetric exposes value on t's benchmark result line (benchstat
// compatible); it is a no-op when t cannot report metrics.
func ReportMetric[X testing.TB](t X, value float64, unit string) {
	if r, ok := unwrap(t).(interface{ ReportMetric(float64, string) }); ok {
		r.ReportMetric(value, unit)
	}
}

// ReportAllocs enables allocation reporting when t supports it.
func ReportAllocs[X testing.TB](t X) {
	if r, ok := unwrap(t).(interface{ ReportAllocs() }); ok {
		r.ReportAllocs()
	}
}

// SetBytes records the number of bytes processed per iteration when t
// supports it.
func SetBytes[X testing.TB](t X, n int64) {
	if r, ok := unwrap(t).(interface{ SetBytes(int64) }); ok {
		r.SetBytes(n)
	}
}

// ResetTimer zeroes the benchmark timer when t supports it (no-op
// otherwise), so generic setup code can keep itself out of the measured
// window without knowing whether it runs under a test or a benchmark.
func ResetTimer[X testing.TB](t X) {
	if r, ok := unwrap(t).(interface{ ResetTimer() }); ok {
		r.ResetTimer()
	}
}

// StartTimer resumes the benchmark timer when t supports it (no-op
// otherwise); pair it with StopTimer around unmeasured teardown work.
func StartTimer[X testing.TB](t X) {
	if r, ok := unwrap(t).(interface{ StartTimer() }); ok {
		r.StartTimer()
	}
}

// StopTimer pauses the benchmark timer when t supports it (no-op
// otherwise), keeping generic teardown work out of the measured window.
func StopTimer[X testing.TB](t X) {
	if r, ok := unwrap(t).(interface{ StopTimer() }); ok {
		r.StopTimer()
	}
}

// Parallel signals that the test may run in parallel with (and only with)
// other parallel tests; benchmarks have no such phase, so it is a no-op
// there (b.RunParallel is a different, intra-benchmark concept).
func Parallel[X testing.TB](t X) {
	if p, ok := unwrap(t).(interface{ Parallel() }); ok {
		p.Parallel()
	}
}
