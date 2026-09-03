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

// RemediationPolicyScope restricts which namespaces this policy applies to.
type RemediationPolicyScope struct {
	// namespaces restricts this policy to the listed namespaces. Omit or leave
	// empty for cluster-wide.
	// +optional
	Namespaces []string `json:"namespaces,omitempty"`
}

// DetectorConfig enables or disables a single detector for this policy.
type DetectorConfig struct {
	// name is the detector being configured.
	// +required
	Name DetectorType `json:"name"`

	// enabled turns this detector on or off within the policy's scope.
	// +optional
	// +kubebuilder:default=true
	Enabled bool `json:"enabled,omitempty"`
}

// RateLimit caps the number of remediation actions allowed within a rolling
// time window, preventing remediation storms during widespread outages.
type RateLimit struct {
	// maxActions is the maximum number of remediation actions permitted within Window.
	// +required
	// +kubebuilder:validation:Minimum=1
	MaxActions int32 `json:"maxActions"`

	// window is the rolling time window MaxActions is evaluated over.
	// +required
	Window metav1.Duration `json:"window"`
}

// CircuitBreakerDefaults defines the default circuit-breaker behavior for
// playbooks bound to this policy: a playbook is auto-disabled after
// FailureThreshold consecutive failed remediations or verification failures,
// requiring manual re-enable.
type CircuitBreakerDefaults struct {
	// failureThreshold is the number of consecutive failed remediations or
	// verification failures that trips the breaker.
	// +optional
	// +kubebuilder:default=3
	// +kubebuilder:validation:Minimum=1
	FailureThreshold int32 `json:"failureThreshold,omitempty"`
}

// RemediationPolicySafety defines global safety limits enforced across every
// playbook bound to this policy, on top of each playbook's own safety settings.
type RemediationPolicySafety struct {
	// dryRun, when true, forces every playbook under this policy into dry-run
	// mode regardless of the playbook's own setting.
	// +optional
	// +kubebuilder:default=true
	DryRun bool `json:"dryRun,omitempty"`

	// globalRateLimit caps total remediation actions across all playbooks
	// bound to this policy.
	// +optional
	GlobalRateLimit *RateLimit `json:"globalRateLimit,omitempty"`

	// blastRadiusPercent is the default per-action blast-radius cap applied
	// when a bound playbook does not set its own maxAffectedPercent.
	// +optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=100
	BlastRadiusPercent *int32 `json:"blastRadiusPercent,omitempty"`

	// circuitBreaker defines the default circuit-breaker behavior for
	// playbooks bound to this policy.
	// +optional
	CircuitBreaker *CircuitBreakerDefaults `json:"circuitBreaker,omitempty"`
}

// RemediationPolicySpec defines the desired state of RemediationPolicy
type RemediationPolicySpec struct {
	// scope restricts which namespaces this policy applies to. Omit for cluster-wide.
	// +optional
	Scope *RemediationPolicyScope `json:"scope,omitempty"`

	// detectors lists which detectors are enabled within this policy's scope.
	// +optional
	Detectors []DetectorConfig `json:"detectors,omitempty"`

	// safety defines global safety limits enforced across every playbook
	// bound to this policy.
	// +optional
	Safety *RemediationPolicySafety `json:"safety,omitempty"`
}

// RemediationPolicyStatus defines the observed state of RemediationPolicy.
type RemediationPolicyStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// For Kubernetes API conventions, see:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	// conditions represent the current state of the RemediationPolicy resource.
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

// RemediationPolicy is the Schema for the remediationpolicies API
type RemediationPolicy struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of RemediationPolicy
	// +required
	Spec RemediationPolicySpec `json:"spec"`

	// status defines the observed state of RemediationPolicy
	// +optional
	Status RemediationPolicyStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// RemediationPolicyList contains a list of RemediationPolicy
type RemediationPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []RemediationPolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RemediationPolicy{}, &RemediationPolicyList{})
}
