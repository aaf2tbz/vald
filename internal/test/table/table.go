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

// Package table is the table-driven core of the test framework: one
// parametric case shape (CaseFor) and one runner (Run) drive both tests and
// benchmarks, with the concrete entry type expressed through capability.Runner[X].
// The ergonomic *testing.T / *testing.B names (Case, BenchmarkCase, ...)
// live in the parent internal/test facade; this package deliberately
// exposes only the parametric API.
package table

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/vdaas/vald/internal/conv"
	"github.com/vdaas/vald/internal/encoding/json"
	"github.com/vdaas/vald/internal/errors"
	"github.com/vdaas/vald/internal/safety"
	"github.com/vdaas/vald/internal/test/capability"
	"github.com/vdaas/vald/internal/test/goleak"
)

// CaseFor is one row of a table-driven test or benchmark: its Args feed the
// do function under test, Want is compared against the outcome (CheckFunc
// defaulting to DefaultCheck), and the optional hooks run around it.
type CaseFor[X capability.Runner[X], T, A any] struct {
	Want       Result[T]
	Args       A
	BeforeFunc BeforeFuncFor[X, A]
	AfterFunc  AfterFuncFor[X, T, A]
	CheckFunc  CheckFuncFor[X, T]
	Name       string
}

// Result carries either a success value or an error, serving as both the
// expected (Want) and actual (got) side of a case's outcome comparison.
type Result[T any] struct {
	Val T
	Err error
}

type (
	// BeforeFuncFor prepares (and may transform) a case's Args before do runs.
	BeforeFuncFor[X capability.Runner[X], A any] func(context.Context, X, A) A
	// AfterFuncFor observes the case outcome for cleanup or extra assertions.
	AfterFuncFor[X capability.Runner[X], T, A any] func(context.Context, X, A, T, error) error
	// CheckFuncFor compares the wanted and actual Result of a case.
	CheckFuncFor[X capability.Runner[X], T any] func(tt X, want, got Result[T]) error
	// DoFor is the function under test, executed once per case.
	DoFor[X capability.Runner[X], T, A any] func(X, A) (T, error)
)

// DefaultCheck is the CheckFunc used when a case does not provide one: the
// error must match errors.Is-wise and the value must be deeply equal, with
// mismatches rendered as JSON where possible for readable diffs.
func DefaultCheck[X capability.Runner[X], T any](tt X, want, got Result[T]) error {
	tt.Helper()
	if !errors.Is(got.Err, want.Err) {
		return errors.Errorf("got_error: \"%#v\",\n\t\t\t\twant: \"%#v\"", got.Err, want.Err)
	}
	if !reflect.DeepEqual(got.Val, want.Val) {
		gb, err := json.Marshal(got.Val)
		gs := conv.Btoa(gb)
		if err != nil || gb == nil {
			gs = fmt.Sprintf("%#v", got.Val)
		}

		wb, err := json.Marshal(want.Val)
		ws := conv.Btoa(wb)
		if err != nil || wb == nil {
			ws = fmt.Sprintf("%#v", want.Val)
		}
		return errors.Errorf("got: \"%s\",\n\t\t\t\twant: \"%s\"", gs, ws)
	}
	return nil
}

// runCase executes a single Case: before hook, do, check and after hook,
// with goroutine-leak verification scoped to the case.
func runCase[X capability.Runner[X], T, A any](
	ctx context.Context, tt X, do DoFor[X, T, A], test CaseFor[X, T, A],
) error {
	tt.Helper()
	defer goleak.VerifyNone(tt, goleak.IgnoreCurrent())
	args := test.Args
	if test.BeforeFunc != nil {
		args = test.BeforeFunc(ctx, tt, args)
	}
	checkFunc := test.CheckFunc
	if checkFunc == nil {
		checkFunc = DefaultCheck[X, T]
	}
	got, err := do(tt, args)
	if err = checkFunc(tt, test.Want, Result[T]{
		Val: got,
		Err: err,
	}); err != nil {
		return err
	}
	if test.AfterFunc != nil {
		err = test.AfterFunc(ctx, tt, args, got, err)
		if err != nil {
			return err
		}
	}
	return nil
}

// Run executes each case as a subtest (or sub-benchmark) of t, recovering
// panics into errors and reporting the first case failure through its
// return value; the subtest itself is not failed, so the caller decides how
// to surface it.
func Run[X capability.Runner[X], T, A any](
	ctx context.Context, t X, do DoFor[X, T, A], tests ...CaseFor[X, T, A],
) error {
	t.Helper()
	ech := make(chan error, len(tests))
	defer close(ech)
	for _, tc := range tests {
		select {
		case err := <-ech:
			if err != nil {
				return err
			}
		case <-ctx.Done():
			err := ctx.Err()
			return err
		default:
			test := tc
			t.Run(test.Name, func(tt X) {
				tt.Helper()
				err := safety.RecoverFunc(func() error {
					return runCase(ctx, tt, do, test)
				})()
				if err != nil {
					select {
					case ech <- err:
					case <-ctx.Done():
						err := ctx.Err()
						tt.Error(err)
					}
				}
			})
		}
	}
	select {
	case err := <-ech:
		return err
	case <-ctx.Done():
		err := ctx.Err()
		return err
	default:
		return nil
	}
}

// The framework must keep accepting the standard testing entries.
var (
	_ capability.Runner[*testing.T] = (*testing.T)(nil)
	_ capability.Runner[*testing.B] = (*testing.B)(nil)
)
