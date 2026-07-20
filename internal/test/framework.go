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

// Package test provides functions for general testing use
package test

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/vdaas/vald/internal/conv"
	"github.com/vdaas/vald/internal/encoding/json"
	"github.com/vdaas/vald/internal/errors"
	"github.com/vdaas/vald/internal/safety"
	"github.com/vdaas/vald/internal/test/goleak"
)

// Runner constrains the concrete testing entry types this framework can
// drive. testing.TB deliberately omits Run (T.Run and B.Run take callbacks
// of their own concrete type, so no single method signature fits the
// interface), which is why the constraint is self-referential: X must both
// behave like testing.TB and spawn subtests of its own type. *testing.T and
// *testing.B satisfy it; *testing.F does not (it has Fuzz, not Run).
type Runner[X testing.TB] interface {
	testing.TB
	Run(name string, f func(X)) bool
}

type CaseFor[X Runner[X], T, A any] struct {
	Want       Result[T]
	Args       A
	BeforeFunc BeforeFuncFor[X, A]
	AfterFunc  AfterFuncFor[X, T, A]
	CheckFunc  CheckFuncFor[X, T]
	Name       string
}

type Result[T any] struct {
	Val T
	Err error
}

type (
	BeforeFuncFor[X Runner[X], A any]   func(context.Context, X, A) A
	AfterFuncFor[X Runner[X], T, A any] func(context.Context, X, A, T, error) error
	CheckFuncFor[X Runner[X], T any]    func(tt X, want, got Result[T]) error
	DoFor[X Runner[X], T, A any]        func(X, A) (T, error)
)

// The historical *testing.T-based names are kept as generic type aliases
// (Go 1.24+) so existing call sites compile unchanged, alongside the
// *testing.B instantiations for table-driven benchmarks.
type (
	Case[T, A any]      = CaseFor[*testing.T, T, A]
	BeforeFunc[A any]   = BeforeFuncFor[*testing.T, A]
	AfterFunc[T, A any] = AfterFuncFor[*testing.T, T, A]
	CheckFunc[T any]    = CheckFuncFor[*testing.T, T]
	Do[T, A any]        = DoFor[*testing.T, T, A]

	BenchmarkCase[T, A any]      = CaseFor[*testing.B, T, A]
	BenchmarkBeforeFunc[A any]   = BeforeFuncFor[*testing.B, A]
	BenchmarkAfterFunc[T, A any] = AfterFuncFor[*testing.B, T, A]
	BenchmarkCheckFunc[T any]    = CheckFuncFor[*testing.B, T]
	BenchmarkDo[T, A any]        = DoFor[*testing.B, T, A]
)

func DefaultCheck[X Runner[X], T any](tt X, want, got Result[T]) error {
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
func runCase[X Runner[X], T, A any](
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

func Run[X Runner[X], T, A any](
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
