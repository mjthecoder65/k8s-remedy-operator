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

// Package detectors defines the pluggable Detector interface and a registry
// of implementations. New failure signatures are added by implementing
// Detector and calling Register from an init() function; the controller
// looks detectors up by name and never needs to change.
package detectors

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	remedyv1alpha1 "github.com/mjthecoder65/k8s-remedy-operator/api/v1alpha1"
)

// DetectionResult identifies a single object that has tripped a detector's
// threshold and is a candidate for remediation.
type DetectionResult struct {
	// ObjectKey identifies the object that tripped detection.
	ObjectKey client.ObjectKey

	// Reason is a short, human-readable explanation of why detection fired,
	// suitable for a Kubernetes Event or audit log entry.
	Reason string
}

// Detector scans objects within a namespace (or cluster-wide, when namespace
// is empty) and reports which ones currently trip the given threshold.
type Detector interface {
	// Name returns the DetectorType this Detector implements.
	Name() remedyv1alpha1.DetectorType

	// Detect returns the set of objects currently tripping threshold. A nil
	// threshold means the detector should apply its own built-in default.
	Detect(ctx context.Context, cli client.Client, namespace string, threshold *remedyv1alpha1.PlaybookThreshold) ([]DetectionResult, error)
}

var registry = map[remedyv1alpha1.DetectorType]Detector{}

// Register adds a Detector to the registry under its own Name(). Intended to
// be called from an implementation package's init() function.
func Register(d Detector) {
	registry[d.Name()] = d
}

// Get looks up a registered Detector by name.
func Get(name remedyv1alpha1.DetectorType) (Detector, error) {
	d, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("no detector registered for %q", name)
	}
	return d, nil
}
