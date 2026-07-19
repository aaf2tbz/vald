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

package x1b

import (
	"io"

	"github.com/vdaas/vald/internal/errors"
)

var (
	// ErrTruncatedData is returned when the input ends before a vector's
	// declared dimension can be fully read, including the case where a
	// trailing dimension header itself is shorter than 4 bytes.
	ErrTruncatedData = errors.New("x1b: truncated vector data")

	// errNotImplemented is returned by every Parse* stub below.
	//
	// TODO(t3-x1b-large-dataset): remove this sentinel and the Parse* stub
	// bodies once the real GREEN-phase implementation lands; see
	// loader_test.go for the RED-phase contract these functions must satisfy.
	errNotImplemented = errors.New("x1b: not implemented")
)

// ParseFvecs decodes fvecs-formatted data read from r: a sequence of records,
// each a 4-byte little-endian int32 dimension header followed by that many
// little-endian float32 elements, repeated until io.EOF.
//
// TODO(t3-x1b-large-dataset): implement; currently a RED-phase stub.
func ParseFvecs(r io.Reader) ([][]float32, error) {
	return nil, errNotImplemented
}

// ParseBvecs decodes bvecs-formatted data read from r: a sequence of records,
// each a 4-byte little-endian int32 dimension header followed by that many
// uint8 elements, repeated until io.EOF.
//
// TODO(t3-x1b-large-dataset): implement; currently a RED-phase stub.
func ParseBvecs(r io.Reader) ([][]uint8, error) {
	return nil, errNotImplemented
}

// ParseIvecs decodes ivecs-formatted data read from r: a sequence of records,
// each a 4-byte little-endian int32 dimension header followed by that many
// little-endian int32 elements, repeated until io.EOF. ivecs files are
// typically used to store groundtruth neighbor indices.
//
// TODO(t3-x1b-large-dataset): implement; currently a RED-phase stub.
func ParseIvecs(r io.Reader) ([][]int32, error) {
	return nil, errNotImplemented
}
