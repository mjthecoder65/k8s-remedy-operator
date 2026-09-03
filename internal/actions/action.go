/*
Copyright 2026.

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

// Package actions defines the pluggable Action interface and a registry of
// implementations. New remediation actions are added by implementing Action
// and calling Register from an init() function; the controller looks actions
// up by name and never needs to change.
package actions

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	remedyv1alpha1 "github.com/mjthecoder65/k8s-remedy-operator/api/v1alpha1"
)

// Input carries everything an Action needs to execute against a single
// target object.
type Input struct {
	// Target identifies the object the action should be applied to.
	Target client.ObjectKey

	// Spec is the action configuration from the triggering Playbook.
	Spec remedyv1alpha1.PlaybookAction

	// DryRun, when true, means the Action must not mutate cluster state: it
	// should determine what it would do and report that via Result instead.
	DryRun bool
}

// Result records what an Action did (or, in dry-run, would have done) so the
// caller can emit events, write an audit log entry, and — later — roll back.
type Result struct {
	// Applied is false when DryRun prevented actual execution.
	Applied bool

	// Message is a human-readable summary of what happened or would happen.
	Message string
}

// Action performs a single remediation action against a target object.
type Action interface {
	// Type returns the ActionType this Action implements.
	Type() remedyv1alpha1.ActionType

	// Execute performs the action, or — when input.DryRun is true —
	// determines and reports what it would do without mutating cluster state.
	Execute(ctx context.Context, cli client.Client, input Input) (Result, error)
}

var registry = map[remedyv1alpha1.ActionType]Action{}

// Register adds an Action to the registry under its own Type(). Intended to
// be called from an implementation package's init() function.
func Register(a Action) {
	registry[a.Type()] = a
}

// Get looks up a registered Action by type.
func Get(actionType remedyv1alpha1.ActionType) (Action, error) {
	a, ok := registry[actionType]
	if !ok {
		return nil, fmt.Errorf("no action registered for %q", actionType)
	}
	return a, nil
}
