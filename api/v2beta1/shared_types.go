package v2beta1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Status - The status for custom resources managed by the operator-sdk.
type Status struct {
	Conditions []Condition `json:"conditions,omitempty"`
	Deployment *string     `json:"deployment,omitempty"`
	Resources  []Resources `json:"resources,omitempty"`
}

// Condition - the condition for the ansible operator
// https://github.com/operator-framework/operator-sdk/blob/master/internal/ansible/controller/status/types.go#L101
type Condition struct {
	Type               ConditionType   `json:"type"`
	Status             ConditionStatus `json:"status"`
	LastTransitionTime metav1.Time     `json:"lastTransitionTime"`
	AnsibleResult      *ResultAnsible  `json:"ansibleResult,omitempty"`
	Reason             string          `json:"reason"`
	Message            string          `json:"message"`
}

// ConditionType specifies a string for field ConditionType
type ConditionType string

// ConditionStatus specifies a string for field ConditionType
type ConditionStatus string

// This const specifies allowed fields for Status
const (
	ConditionTrue    ConditionStatus = "True"
	ConditionFalse   ConditionStatus = "False"
	ConditionUnknown ConditionStatus = "Unknown"
)

// ResultAnsible - encapsulation of the ansible result. 'AnsibleResult' is turned around in struct to comply with linting
type ResultAnsible struct {
	Ok               int    `json:"ok"`
	Changed          int    `json:"changed"`
	Skipped          int    `json:"skipped"`
	Failures         int    `json:"failures"`
	TimeOfCompletion string `json:"completion"`
}

// Resources is the struct for the resources field within status
type Resources struct {
	APIVersion *string `json:"apiversion,omitempty"`
	Kind       *string `json:"kind,omitempty"`
	Name       *string `json:"name,omitempty"`
}

// General is the struct with all generic fields for the crds
type General struct {
	Dataset        string  `json:"dataset"`
	Theme          *string `json:"theme,omitempty"`
	DatasetOwner   string  `json:"datasetOwner"`
	ServiceVersion *string `json:"serviceVersion,omitempty"`
	DataVersion    *string `json:"dataVersion,omitempty"`
}

// Kubernetes is the struct with all fields that can be defined in kubernetes fields in the crds
type Kubernetes struct {
	Autoscaling *Autoscaling                 `json:"autoscaling,omitempty"`
	HealthCheck *HealthCheck                 `json:"healthCheck,omitempty"`
	Resources   *corev1.ResourceRequirements `json:"resources,omitempty"`
	Lifecycle   *Lifecycle                   `json:"lifecycle,omitempty"`
}

// Autoscaling is the struct with all fields to configure autoscalers for the crs
type Autoscaling struct {
	AverageCPUUtilization *int `json:"averageCpuUtilization,omitempty"`
	MinReplicas           *int `json:"minReplicas,omitempty"`
	MaxReplicas           *int `json:"maxReplicas,omitempty"`
}

// HealthCheck is the struct with all fields to configure healthchecks for the crs
type HealthCheck struct {
	Querystring *string `json:"querystring,omitempty"`
	Mimetype    *string `json:"mimetype,omitempty"`
	Boundingbox *string `json:"boundingbox,omitempty"`
}

// Lifecycle is the struct with the fields to configure lifecycle settings for the resources
type Lifecycle struct {
	TTLInDays *int `json:"ttlInDays,omitempty"`
}

// WMSWFSOptions is the struct with options available in the operator
type WMSWFSOptions struct {
	// +kubebuilder:default:=true
	IncludeIngress bool `json:"includeIngress"`
	// +kubebuilder:default:=true
	AutomaticCasing bool `json:"automaticCasing"`
	// +kubebuilder:default:=true
	ValidateRequests         *bool `json:"validateRequests,omitempty"`
	RewriteGroupToDataLayers *bool `json:"rewriteGroupToDataLayers,omitempty"`
	DisableWebserviceProxy   *bool `json:"disableWebserviceProxy,omitempty"`
	// +kubebuilder:default:=true
	PrefetchData                *bool `json:"prefetchData,omitempty"`
	ValidateChildStyleNameEqual *bool `json:"validateChildStyleNameEqual,omitempty"`
}

// Authority is a struct for the authority fields in WMS and WFS crds
type Authority struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// Data is a struct for the data field for a WMSLayer or WFS FeatureType
type Data struct {
	GPKG    *GPKG    `json:"gpkg,omitempty"`
	Postgis *Postgis `json:"postgis,omitempty"`
	Tif     *Tif     `json:"tif,omitempty"`
}

// GPKG is a struct for the gpkg field for a WMSLayer or WFS FeatureType
type GPKG struct {
	BlobKey      string   `json:"blobKey"`
	Table        string   `json:"table"`
	GeometryType string   `json:"geometryType"`
	Columns      []string `json:"columns"`
	// In a new version Aliases should become part of Columns
	Aliases map[string]string `json:"aliases,omitempty"`
}

// Postgis is a struct for the Postgis db config for a WMSLayer or WFS FeatureType
// connection details are passed through the environment
type Postgis struct {
	Table        string   `json:"table"`
	GeometryType string   `json:"geometryType"`
	Columns      []string `json:"columns"`
	// In a new version Aliases should become part of Columns
	Aliases map[string]string `json:"aliases,omitempty"`
}

// Tif is a struct for the Tif field for a WMSLayer
type Tif struct {
	BlobKey                     string  `json:"blobKey"`
	GetFeatureInfoIncludesClass *bool   `json:"getFeatureInfoIncludesClass,omitempty"`
	Offsite                     *string `json:"offsite,omitempty"`
	Resample                    *string `json:"resample,omitempty"`
}
