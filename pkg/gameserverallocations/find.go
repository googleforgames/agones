// Copyright 2018 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gameserverallocations

// packedComparator prioritises Nodes with GameServers that are allocated, and then Nodes with the most
// Ready GameServers -- this will bin pack allocated game servers together.
func packedComparator(bestCount, currentCount NodeCount) bool {
	if currentCount.allocated == bestCount.allocated && currentCount.ready > bestCount.ready {
		return true
	} else if currentCount.allocated > bestCount.allocated {
		return true
	}

	return false
}

// distributedComparator is the inverse of the packed comparator,
// looking to distribute allocated gameservers on as many nodes as possible.
func distributedComparator(bestCount, currentCount NodeCount) bool {
	return !packedComparator(bestCount, currentCount)
}
