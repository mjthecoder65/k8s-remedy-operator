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

package restart

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	remedyv1alpha1 "github.com/mjthecoder65/k8s-remedy-operator/api/v1alpha1"
	"github.com/mjthecoder65/k8s-remedy-operator/internal/actions"
)

func newPod() *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "crashy", Namespace: "default"},
	}
}

func TestExecute_DryRunDoesNotDeletePod(t *testing.T) {
	pod := newPod()
	cli := fake.NewClientBuilder().WithObjects(pod).Build()

	result, err := (Action{}).Execute(context.Background(), cli, actions.Input{
		Target: client.ObjectKeyFromObject(pod),
		Spec:   remedyv1alpha1.PlaybookAction{Type: remedyv1alpha1.ActionRestart},
		DryRun: true,
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if result.Applied {
		t.Errorf("expected Applied=false in dry-run, got true")
	}

	var got corev1.Pod
	if err := cli.Get(context.Background(), client.ObjectKeyFromObject(pod), &got); err != nil {
		t.Errorf("pod should still exist after dry-run, but Get failed: %v", err)
	}
}

func TestExecute_DeletesPodWhenNotDryRun(t *testing.T) {
	pod := newPod()
	cli := fake.NewClientBuilder().WithObjects(pod).Build()

	result, err := (Action{}).Execute(context.Background(), cli, actions.Input{
		Target: client.ObjectKeyFromObject(pod),
		Spec:   remedyv1alpha1.PlaybookAction{Type: remedyv1alpha1.ActionRestart},
		DryRun: false,
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !result.Applied {
		t.Errorf("expected Applied=true, got false")
	}

	err = cli.Get(context.Background(), client.ObjectKeyFromObject(pod), &corev1.Pod{})
	if !apierrors.IsNotFound(err) {
		t.Errorf("expected pod to be deleted, Get returned err=%v", err)
	}
}

func TestExecute_MissingPodReturnsNotFound(t *testing.T) {
	cli := fake.NewClientBuilder().Build()

	_, err := (Action{}).Execute(context.Background(), cli, actions.Input{
		Target: client.ObjectKey{Name: "does-not-exist", Namespace: "default"},
		Spec:   remedyv1alpha1.PlaybookAction{Type: remedyv1alpha1.ActionRestart},
	})
	if !apierrors.IsNotFound(err) {
		t.Errorf("expected a NotFound error, got %v", err)
	}
}
