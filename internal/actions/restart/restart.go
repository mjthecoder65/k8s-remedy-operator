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

// Package restart implements the restart Action: it deletes a Pod so its
// owning controller (Deployment, ReplicaSet, StatefulSet, ...) recreates it.
package restart

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	remedyv1alpha1 "github.com/mjthecoder65/k8s-remedy-operator/api/v1alpha1"
	"github.com/mjthecoder65/k8s-remedy-operator/internal/actions"
)

// Action implements actions.Action for the restart type.
type Action struct{}

// Type implements actions.Action.
func (Action) Type() remedyv1alpha1.ActionType {
	return remedyv1alpha1.ActionRestart
}

// Execute implements actions.Action.
func (Action) Execute(ctx context.Context, cli client.Client, input actions.Input) (actions.Result, error) {
	var pod corev1.Pod
	if err := cli.Get(ctx, input.Target, &pod); err != nil {
		return actions.Result{}, fmt.Errorf("getting pod %s: %w", input.Target, err)
	}

	msg := fmt.Sprintf("delete pod %s to force restart", input.Target)
	if input.DryRun {
		return actions.Result{Applied: false, Message: "[dry-run] would " + msg}, nil
	}

	if err := cli.Delete(ctx, &pod); err != nil {
		return actions.Result{}, fmt.Errorf("deleting pod %s: %w", input.Target, err)
	}
	return actions.Result{Applied: true, Message: msg}, nil
}

func init() {
	actions.Register(Action{})
}
