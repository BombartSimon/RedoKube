package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// OpenAPISpecSpec defines the desired state of OpenAPISpec
type OpenAPISpecSpec struct {
	// Title of the API documentation
	Title string `json:"title"`

	// Path to the OpenAPI specification file or URL
	// +optional if specContent is provided
	SpecPath string `json:"specPath,omitempty"`

	// Direct OpenAPI specification content in JSON or YAML format
	// +optional if specPath is provided
	SpecContent string `json:"specContent,omitempty"`

	// Optional description for the API
	Description string `json:"description,omitempty"`

	// Version of the API documentation
	Version string `json:"version,omitempty"`

	// When enabled, generates fake examples for the OpenAPI specification
	// +optional
	Mock bool `json:"mock,omitempty"`

	// Theme customization options for Redoc
	Theme map[string]string `json:"theme,omitempty"`
}

// OpenAPISpecStatus defines the observed state of OpenAPISpec
type OpenAPISpecStatus struct {
	// Represents the current state of the OpenAPISpec
	// +kubebuilder:validation:Enum=Pending;Available;Failed
	Status string `json:"status"`

	// URL where the documentation is available
	URL string `json:"url,omitempty"`

	// Last time the documentation was updated
	LastUpdated metav1.Time `json:"lastUpdated,omitempty"`

	// Error message in case of failure
	ErrorMessage string `json:"errorMessage,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.status"
//+kubebuilder:printcolumn:name="URL",type="string",JSONPath=".status.url"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// OpenAPISpec is the Schema for the openapispec API
type OpenAPISpec struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OpenAPISpecSpec   `json:"spec,omitempty"`
	Status OpenAPISpecStatus `json:"status,omitempty"`
}

// DeepCopyInto copies all properties of this object into another object of the same type
func (in *OpenAPISpec) DeepCopyInto(out *OpenAPISpec) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)

	// Copy spec
	out.Spec = OpenAPISpecSpec{
		Title:       in.Spec.Title,
		SpecPath:    in.Spec.SpecPath,
		SpecContent: in.Spec.SpecContent,
		Description: in.Spec.Description,
		Version:     in.Spec.Version,
		Mock:        in.Spec.Mock,
	}

	if in.Spec.Theme != nil {
		out.Spec.Theme = make(map[string]string)
		for k, v := range in.Spec.Theme {
			out.Spec.Theme[k] = v
		}
	}

	// Copy status
	out.Status = OpenAPISpecStatus{
		Status:       in.Status.Status,
		URL:          in.Status.URL,
		ErrorMessage: in.Status.ErrorMessage,
	}

	if !in.Status.LastUpdated.IsZero() {
		out.Status.LastUpdated = *in.Status.LastUpdated.DeepCopy()
	}
}

// DeepCopy returns a deep copy of this OpenAPISpec
func (in *OpenAPISpec) DeepCopy() *OpenAPISpec {
	if in == nil {
		return nil
	}
	out := new(OpenAPISpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject returns a deep copy as an Object
func (in *OpenAPISpec) DeepCopyObject() runtime.Object {
	return in.DeepCopy()
}

//+kubebuilder:object:root=true

// OpenAPISpecList contains a list of OpenAPISpec
type OpenAPISpecList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OpenAPISpec `json:"items"`
}

// DeepCopyInto copies all properties of this object into another object of the same type
func (in *OpenAPISpecList) DeepCopyInto(out *OpenAPISpecList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)

	if in.Items != nil {
		out.Items = make([]OpenAPISpec, len(in.Items))
		for i := range in.Items {
			in.Items[i].DeepCopyInto(&out.Items[i])
		}
	}
}

// DeepCopy returns a deep copy of this OpenAPISpecList
func (in *OpenAPISpecList) DeepCopy() *OpenAPISpecList {
	if in == nil {
		return nil
	}
	out := new(OpenAPISpecList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject returns a deep copy as an Object
func (in *OpenAPISpecList) DeepCopyObject() runtime.Object {
	return in.DeepCopy()
}

func init() {
	SchemeBuilder.Register(&OpenAPISpec{}, &OpenAPISpecList{})
}
