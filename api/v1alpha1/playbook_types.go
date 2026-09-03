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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DetectorType identifies which detector a playbook trigger listens to.
// +kubebuilder:validation:Enum=crashloop;memorypressure;certexpiry;dependencytimeout
type DetectorType string

const (
	DetectorCrashLoop         DetectorType = "crashloop"
	DetectorMemoryPressure    DetectorType = "memorypressure"
	DetectorCertExpiry        DetectorType = "certexpiry"
	DetectorDependencyTimeout DetectorType = "dependencytimeout"
)

// ActionType identifies which built-in remediation action a playbook executes.
// +kubebuilder:validation:Enum=restart;rollback;scale;certrenew;cordon
type ActionType string

const (
	ActionRestart   ActionType = "restart"
	ActionRollback  ActionType = "rollback"
	ActionScale     ActionType = "scale"
	ActionCertRenew ActionType = "certrenew"
	ActionCordon    ActionType = "cordon"
)

// HealthCheckType identifies how post-action verification is performed.
// +kubebuilder:validation:Enum=readinessProbe;custom
type HealthCheckType string

const (
	HealthCheckReadinessProbe HealthCheckType = "readinessProbe"
	HealthCheckCustom         HealthCheckType = "custom"
)

// PlaybookThreshold defines the detector-specific condition that must be met
// for a trigger to fire.
type PlaybookThreshold struct {
	// restartCount is the number of container restarts within Window that
	// trips the crashloop detector.
	// +optional
	// +kubebuilder:validation:Minimum=1
	RestartCount *int32 `json:"restartCount,omitempty"`

	// window is the rolling time window over which RestartCount (or an
	// equivalent detector-specific counter) is evaluated.
	// +optional
	Window *metav1.Duration `json:"window,omitempty"`
}

// PlaybookTrigger defines which detector activates this playbook and under
// what threshold.
type PlaybookTrigger struct {
	// detector is the name of the registered Detector this playbook responds to.
	// +required
	Detector DetectorType `json:"detector"`

	// threshold is the detector-specific condition that must be met to fire.
	// +optional
	Threshold *PlaybookThreshold `json:"threshold,omitempty"`
}

// PlaybookAction defines the remediation action executed when the trigger fires.
type PlaybookAction struct {
	// type is the built-in or plugin action to execute.
	// +required
	Type ActionType `json:"type"`

	// maxAffectedPercent caps the percentage of a workload's replicas this
	// action may touch in a single remediation window (blast-radius cap).
	// +optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=100
	MaxAffectedPercent *int32 `json:"maxAffectedPercent,omitempty"`
}

// PlaybookVerification defines the post-action health check that confirms a
// remediation succeeded.
type PlaybookVerification struct {
	// healthCheck selects how post-action health is determined.
	// +optional
	// +kubebuilder:default=readinessProbe
	HealthCheck HealthCheckType `json:"healthCheck,omitempty"`

	// timeout is how long to wait for the health check to pass before the
	// remediation is considered failed and eligible for rollback.
	// +optional
	Timeout *metav1.Duration `json:"timeout,omitempty"`
}

// PlaybookSafety defines per-playbook safety overrides. A cluster-wide
// --dry-run flag on the manager takes precedence over DryRun=false here.
type PlaybookSafety struct {
	// dryRun logs and emits an event for the action this playbook would take,
	// without executing it.
	// +optional
	// +kubebuilder:default=false
	DryRun bool `json:"dryRun,omitempty"`

	// requireApproval gates execution of this playbook's action behind manual
	// approval (annotation, Slack, or webhook).
	// +optional
	// +kubebuilder:default=false
	RequireApproval bool `json:"requireApproval,omitempty"`
}

// PlaybookSpec defines the desired state of Playbook
type PlaybookSpec struct {
	// trigger defines which detector activates this playbook and under what threshold.
	// +required
	Trigger PlaybookTrigger `json:"trigger"`

	// action defines the remediation action executed when the trigger fires.
	// +required
	Action PlaybookAction `json:"action"`

	// verification defines the post-action health check that confirms the
	// remediation succeeded.
	// +optional
	Verification *PlaybookVerification `json:"verification,omitempty"`

	// safety defines per-playbook safety overrides.
	// +optional
	Safety *PlaybookSafety `json:"safety,omitempty"`
}

// PlaybookStatus defines the observed state of Playbook.
type PlaybookStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// For Kubernetes API conventions, see:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	// conditions represent the current state of the Playbook resource.
	// Each condition has a unique type and reflects the status of a specific aspect of the resource.
	//
	// Standard condition types include:
	// - "Available": the resource is fully functional
	// - "Progressing": the resource is being created or updated
	// - "Degraded": the resource failed to reach or maintain its desired state
	//
	// The status of each condition is one of True, False, or Unknown.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Playbook is the Schema for the playbooks API
type Playbook struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of Playbook
	// +required
	Spec PlaybookSpec `json:"spec"`

	// status defines the observed state of Playbook
	// +optional
	Status PlaybookStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// PlaybookList contains a list of Playbook
type PlaybookList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []Playbook `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Playbook{}, &PlaybookList{})
}
