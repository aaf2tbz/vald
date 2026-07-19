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

package metrics

// CalcRecall reports the recall@k score of an approximate nearest-neighbor
// search result against the ground-truth neighbor IDs for a single query.
//
// got is the ID list returned by the (approximate) search, ordered from the
// nearest neighbor to the farthest. truth is the corresponding ground-truth
// neighbor ID list for the same query, e.g. one row of
// tests/v2/e2e/hdf5.Dataset.Neighbors (also ordered nearest to farthest, and
// typically longer than the k actually requested from the search, since
// ann-benchmarks-style hdf5 files usually store the top-100 ground truth
// regardless of the benchmark's k).
//
// recall@k depends only on set membership, not on order:
//
//	effectiveK := min(k, len(truth))
//	recall@k   := |top(got, effectiveK) ∩ top(truth, effectiveK)| / effectiveK
//
// where top(x, n) denotes the first min(n, len(x)) elements of x. In other
// words, both got and truth are truncated down to effectiveK entries before
// being compared as unordered sets — a matching ID that only appears beyond
// position effectiveK in got (e.g. because the search returned more than k
// results) must NOT count towards the score, and ground-truth entries beyond
// effectiveK must NOT be considered "correct" either.
//
// If effectiveK <= 0 (k <= 0, or truth is empty/nil — e.g. no ground-truth
// dataset is available), CalcRecall returns 0 without an error, since recall
// is undefined for an empty ground truth and 0 is the conservative answer.
//
// This is an independent implementation of the same recall@k definition used
// by the non-exported calcRecall in
// pkg/tools/benchmark/job/service/job.go:309 (linear/brute-force result IDs
// vs. approximate search result IDs, intersection count divided by the
// ground-truth count) generalized with an explicit k so callers can compare
// against hdf5 ground-truth rows that are longer than the benchmark's k.
func CalcRecall(got, truth []int, k int) (recall float64) {
	effectiveK := min(len(truth), k)
	if effectiveK <= 0 {
		return 0
	}
	if len(got) > effectiveK {
		got = got[:effectiveK]
	}
	truth = truth[:effectiveK]

	truthIDs := make(map[int]struct{}, effectiveK)
	for _, id := range truth {
		truthIDs[id] = struct{}{}
	}

	var matched float64
	for _, id := range got {
		if _, ok := truthIDs[id]; ok {
			matched++
		}
	}
	return matched / float64(effectiveK)
}
