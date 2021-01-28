// Copyright 2021 Solarwinds Inc.
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

package papertrailgo

import "sync"

// loopFactor is used to help with managing loops
type loopFactor struct {
	track bool
	m     *sync.Mutex
}

// newLoopFactor helps with creating a loopFactor pointer instance
func newLoopFactor(x bool) *loopFactor {
	return &loopFactor{
		track: x,
		m:     new(sync.Mutex),
	}
}

// getBool returns the current boolean value of loopFactor
func (l *loopFactor) getBool() bool {
	l.m.Lock()
	defer l.m.Unlock()
	return l.track
}

// setBool sets the boolean value of loopFactor
func (l *loopFactor) setBool(x bool) {
	l.m.Lock()
	defer l.m.Unlock()
	l.track = x
}
