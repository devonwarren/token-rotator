/*
Copyright 2026 Devon Warren.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package rotation contains helpers shared across per-source token
// reconcilers: schedule evaluation, status condition helpers, and Secret
// export.
package rotation

import (
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
)

var cronParser = cron.NewParser(
	cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow,
)

// Decision describes whether a rotation should happen on this reconcile and
// when the next one is due.
type Decision struct {
	Due     bool
	NextRun time.Time
}

// Evaluate decides whether a rotation is due based on the spec's cron
// schedule, the ForceNow flag, and the last rotation time. A nil
// lastRotation means "never rotated" — rotation is always due.
func Evaluate(
	schedule string, forceNow bool, lastRotation *time.Time, now time.Time,
) (Decision, error) {
	sched, err := cronParser.Parse(schedule)
	if err != nil {
		return Decision{}, fmt.Errorf("parse rotation schedule %q: %w", schedule, err)
	}

	if forceNow || lastRotation == nil {
		return Decision{Due: true, NextRun: sched.Next(now)}, nil
	}

	next := sched.Next(*lastRotation)
	return Decision{Due: !now.Before(next), NextRun: next}, nil
}
