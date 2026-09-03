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

package crashloop

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	remedyv1alpha1 "github.com/mjthecoder65/k8s-remedy-operator/api/v1alpha1"
)

func int32Ptr(i int32) *int32 { return &i }

func podWithRestarts(name string, restarts int32, lastFinished *time.Time, waitingReason string) *corev1.Pod {
	status := corev1.ContainerStatus{
		Name:         "app",
		RestartCount: restarts,
	}
	if lastFinished != nil {
		status.LastTerminationState.Terminated = &corev1.ContainerStateTerminated{
			FinishedAt: metav1.NewTime(*lastFinished),
		}
	}
	if waitingReason != "" {
		status.State.Waiting = &corev1.ContainerStateWaiting{Reason: waitingReason}
	}
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"},
		Status:     corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{status}},
	}
}

func TestDetect_FlagsPodsAtOrAboveThreshold(t *testing.T) {
	fixedNow := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	recent := fixedNow.Add(-1 * time.Minute)

	tests := []struct {
		name      string
		pod       *corev1.Pod
		threshold *remedyv1alpha1.PlaybookThreshold
		want      bool
	}{
		{
			name: "at threshold with recent restart -> flagged",
			pod:  podWithRestarts("a", 5, &recent, ""),
			threshold: &remedyv1alpha1.PlaybookThreshold{
				RestartCount: int32Ptr(5),
				Window:       &metav1.Duration{Duration: 10 * time.Minute},
			},
			want: true,
		},
		{
			name: "below threshold -> not flagged",
			pod:  podWithRestarts("b", 4, &recent, ""),
			threshold: &remedyv1alpha1.PlaybookThreshold{
				RestartCount: int32Ptr(5),
				Window:       &metav1.Duration{Duration: 10 * time.Minute},
			},
			want: false,
		},
		{
			name: "above threshold but restart outside window -> not flagged",
			pod:  podWithRestarts("c", 10, timePtr(fixedNow.Add(-1*time.Hour)), ""),
			threshold: &remedyv1alpha1.PlaybookThreshold{
				RestartCount: int32Ptr(5),
				Window:       &metav1.Duration{Duration: 10 * time.Minute},
			},
			want: false,
		},
		{
			name: "no terminated timestamp but actively in CrashLoopBackOff -> flagged",
			pod:  podWithRestarts("d", 5, nil, "CrashLoopBackOff"),
			threshold: &remedyv1alpha1.PlaybookThreshold{
				RestartCount: int32Ptr(5),
				Window:       &metav1.Duration{Duration: 10 * time.Minute},
			},
			want: true,
		},
		{
			name:      "nil threshold falls back to default restart count",
			pod:       podWithRestarts("e", defaultRestartCount, &recent, ""),
			threshold: nil,
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := fake.NewClientBuilder().WithObjects(tt.pod).Build()
			d := Detector{Now: func() time.Time { return fixedNow }}

			results, err := d.Detect(context.Background(), cli, "default", tt.threshold)
			if err != nil {
				t.Fatalf("Detect returned error: %v", err)
			}

			got := len(results) == 1 && results[0].ObjectKey.Name == tt.pod.Name
			if got != tt.want {
				t.Errorf("pod %q flagged=%v, want %v (results: %+v)", tt.pod.Name, got, tt.want, results)
			}
		})
	}
}

func timePtr(t time.Time) *time.Time { return &t }
