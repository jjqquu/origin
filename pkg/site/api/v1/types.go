package v1

import (
	"k8s.io/kubernetes/pkg/api/unversioned"
	kapi "k8s.io/kubernetes/pkg/api/v1"
)

const (
	// These are internal finalizer values to Origin
	FinalizerOrigin kapi.FinalizerName = "openshift.io/origin"
)

// Address of a site
type SiteAddress struct {
	// URL to access the site
	Url string `json:"url" protobuf:"bytes,1,opt,name=url"`
}

const (
	// StringSourceEncryptedBlockType is the PEM block type used to store an encrypted string
	StringSourceEncryptedBlockType = "ENCRYPTED STRING"
	// StringSourceKeyBlockType is the PEM block type used to store an encrypting key
	StringSourceKeyBlockType = "ENCRYPTING KEY"
)

// StringSource allows specifying a string inline, or externally via env var or file.
// When it contains only a string value, it marshals to a simple JSON string.
type StringSource struct {
	// StringSourceSpec specifies the string value, or external location
	StringSourceSpec `json:"stringSourceSpec,omitempty" protobuf:"bytes,1,opt,name=stringSourceSpec"`
}

// StringSourceSpec specifies a string value, or external location
type StringSourceSpec struct {
	// Value specifies the cleartext value, or an encrypted value if keyFile is specified.
	Value string `json:"value,omitempty" protobuf:"bytes,1,opt,name=value"`

	// Env specifies an envvar containing the cleartext value, or an encrypted value if the keyFile is specified.
	Env string `json:"env,omitempty" protobuf:"bytes,2,opt,name=env"`

	// File references a file containing the cleartext value, or an encrypted value if a keyFile is specified.
	File string `json:"file,omitempty" protobuf:"bytes,3,opt,name=file"`

	// KeyFile references a file containing the key to use to decrypt the value.
	KeyFile string `json:"keyFile,omitempty" protobuf:"bytes,4,opt,name=keyFile"`
}

type SiteType string

// These are the valid phases of a site.
const (
	// Sites in normal status that can accept workloads
	SiteLocal SiteType = "local"
	// Newly registered sites or sites suspended by admin for various reasons. They are not eligible for accepting workloads
	SiteMarathon SiteType = "marathon"
	// Sites in normal status that can accept workloads
	SiteK8s SiteType = "k8s"
	// Sites in normal status that can accept workloads
	SiteOpenshift3 SiteType = "openshift"
)

// SiteSpec describes the attributes on a Site.
type SiteSpec struct {
	// Type of the site
	Type SiteType `json:"type,omitempty" protobuf:"bytes,1,opt,name=type"`
	// Address of the site
	Address SiteAddress `json:"address" protobuf:"bytes,2,opt,name=address"`
	// The credential used to access site. It’s used for system routines (not behalf of users)
	Credential StringSource `json:"credential" protobuf:"bytes,3,opt,name=credential"`

	// Finalizers is an opaque list of values that must be empty to permanently remove object from storage
	Finalizers []kapi.FinalizerName `json:"finalizers,omitempty" protobuf:"bytes,4,rep,name=finalizers,casttype=k8s.io/kubernetes/pkg/api/v1.FinalizerName"`
}

type SitePhase string

// These are the valid phases of a site.
const (
	// Newly registered sites or sites suspended by admin for various reasons. They are not eligible for accepting workloads
	SitePending SitePhase = "pending"
	// Sites in normal status that can accept workloads
	SiteRunning SitePhase = "running"
	// Sites temporarily down or not reachable
	SiteOffline SitePhase = "offline"
	// Sites removed from federation
	SiteTerminated SitePhase = "terminated"
)

// SiteStatus is information about the current status of a site.
type SiteStatus struct {
	// Phase is the recently observed lifecycle phase of the site.
	Phase SitePhase `json:"phase,omitempty" protobuf:"bytes,1,opt,name=phase"`
	// SiteMeta is the meta data of site
	SiteMeta string `json:",inline" protobuf:"bytes,2,opt,name=siteMeta"`
	// SiteAgentAddress is the site agent host name
	SiteAgentAddress string `json:"siteAgentAddress,omitempty" protobuf:"bytes,3,opt,name=siteAgentAddress"`
}

// Site information in Ubernetes
type Site struct {
	unversioned.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#metadata
	kapi.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Spec defines the behavior of the Site.
	Spec SiteSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
	// Status describes the current status of a Site
	Status SiteStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// A list of Sites
type SiteList struct {
	unversioned.TypeMeta `json:",inline"`
	// Standard list metadata.
	// More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#types-kinds
	unversioned.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// List of Site objects.
	Items []Site `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// These constants represent annotations keys affixed to sites
const (
	// SiteDisplayName is an annotation that stores the name displayed when querying for sites
	SiteDisplayName = "openshift.io/display-name"
	// SiteDescription is an annotatoion that holds the description of the site
	SiteDescription = "openshift.io/description"
)
