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

package controller

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	remedyv1alpha1 "github.com/mjthecoder65/k8s-remedy-operator/api/v1alpha1"
	"github.com/mjthecoder65/k8s-remedy-operator/internal/actions"
	"github.com/mjthecoder65/k8s-remedy-operator/internal/detectors"
)

// pollInterval is how often a Playbook is re-evaluated. Phase 1 polls on a
// fixed interval rather than watching the target objects directly; a
// detector-specific Watch can replace this later without changing the
// Detector/Action interfaces.
const pollInterval = 30 * time.Second

// PlaybookReconciler reconciles a Playbook object
type PlaybookReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=remedy.io,resources=playbooks,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=remedy.io,resources=playbooks/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=remedy.io,resources=playbooks/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// Reconcile evaluates a Playbook's trigger against current cluster state and,
// when it fires, executes the configured action - subject to the playbook's
// dry-run setting.
func (r *PlaybookReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	var playbook remedyv1alpha1.Playbook
	if err := r.Get(ctx, req.NamespacedName, &playbook); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	detector, err := detectors.Get(playbook.Spec.Trigger.Detector)
	if err != nil {
		log.Error(err, "cannot evaluate trigger")
		r.Recorder.Event(&playbook, corev1.EventTypeWarning, "DetectorNotFound", err.Error())
		return ctrl.Result{}, nil
	}

	results, err := detector.Detect(ctx, r.Client, playbook.Namespace, playbook.Spec.Trigger.Threshold)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("detecting: %w", err)
	}
	if len(results) == 0 {
		return ctrl.Result{RequeueAfter: pollInterval}, nil
	}

	action, err := actions.Get(playbook.Spec.Action.Type)
	if err != nil {
		log.Error(err, "cannot execute action")
		r.Recorder.Event(&playbook, corev1.EventTypeWarning, "ActionNotFound", err.Error())
		return ctrl.Result{}, nil
	}

	dryRun := playbook.Spec.Safety != nil && playbook.Spec.Safety.DryRun

	for _, detection := range results {
		result, err := action.Execute(ctx, r.Client, actions.Input{
			Target: detection.ObjectKey,
			Spec:   playbook.Spec.Action,
			DryRun: dryRun,
		})
		if err != nil {
			if apierrors.IsNotFound(err) {
				continue // target already gone, nothing to remediate
			}
			log.Error(err, "remediation action failed", "target", detection.ObjectKey, "reason", detection.Reason)
			r.Recorder.Eventf(&playbook, corev1.EventTypeWarning, "RemediationFailed",
				"%s: %s: %v", detection.ObjectKey, detection.Reason, err)
			continue
		}

		reason := "RemediationApplied"
		if !result.Applied {
			reason = "RemediationSkippedDryRun"
		}
		r.Recorder.Eventf(&playbook, corev1.EventTypeNormal, reason,
			"%s: %s: %s", detection.ObjectKey, detection.Reason, result.Message)
	}

	return ctrl.Result{RequeueAfter: pollInterval}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PlaybookReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&remedyv1alpha1.Playbook{}).
		Named("playbook").
		Complete(r)
}
