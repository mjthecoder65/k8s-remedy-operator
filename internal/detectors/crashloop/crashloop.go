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

// Package crashloop implements the crash-loop Detector: it flags Pods whose
// containers have restarted at least as many times as the configured
// threshold, and whose most recent restart falls within the configured
// window.
package crashloop

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	remedyv1alpha1 "github.com/mjthecoder65/k8s-remedy-operator/api/v1alpha1"
	"github.com/mjthecoder65/k8s-remedy-operator/internal/detectors"
)

// defaultRestartCount is used when a Playbook's trigger sets no threshold.
const defaultRestartCount = int32(5)

// Detector implements detectors.Detector for the crashloop signature.
type Detector struct {
	// Now returns the current time; overridable in tests.
	Now func() time.Time
}

func (d Detector) now() time.Time {
	if d.Now != nil {
		return d.Now()
	}
	return time.Now()
}

// Name implements detectors.Detector.
func (Detector) Name() remedyv1alpha1.DetectorType {
	return remedyv1alpha1.DetectorCrashLoop
}

// Detect implements detectors.Detector.
func (d Detector) Detect(ctx context.Context, cli client.Client, namespace string, threshold *remedyv1alpha1.PlaybookThreshold) ([]detectors.DetectionResult, error) {
	restartCount := defaultRestartCount
	var window time.Duration
	if threshold != nil {
		if threshold.RestartCount != nil {
			restartCount = *threshold.RestartCount
		}
		if threshold.Window != nil {
			window = threshold.Window.Duration
		}
	}

	var listOpts []client.ListOption
	if namespace != "" {
		listOpts = append(listOpts, client.InNamespace(namespace))
	}

	var pods corev1.PodList
	if err := cli.List(ctx, &pods, listOpts...); err != nil {
		return nil, fmt.Errorf("listing pods: %w", err)
	}

	now := d.now()
	var results []detectors.DetectionResult
	for i := range pods.Items {
		pod := &pods.Items[i]
		for _, cs := range pod.Status.ContainerStatuses {
			if cs.RestartCount < restartCount {
				continue
			}
			if window > 0 && !recentlyRestarted(cs, now, window) {
				continue
			}
			results = append(results, detectors.DetectionResult{
				ObjectKey: client.ObjectKeyFromObject(pod),
				Reason: fmt.Sprintf("container %q restarted %d times (threshold %d)",
					cs.Name, cs.RestartCount, restartCount),
			})
			break // one detection per pod is enough
		}
	}
	return results, nil
}

// recentlyRestarted reports whether cs's most recent restart happened within
// window of now. Kubelet's own CrashLoopBackOff waiting reason is treated as
// evidence of an imminent/ongoing restart when no terminated timestamp is
// recorded yet.
func recentlyRestarted(cs corev1.ContainerStatus, now time.Time, window time.Duration) bool {
	if cs.LastTerminationState.Terminated != nil {
		return now.Sub(cs.LastTerminationState.Terminated.FinishedAt.Time) <= window
	}
	return cs.State.Waiting != nil && cs.State.Waiting.Reason == "CrashLoopBackOff"
}

func init() {
	detectors.Register(Detector{})
}
