package v1beta3

import (
	"k8s.io/kubernetes/pkg/api/unversioned"
	kapi "k8s.io/kubernetes/pkg/api/v1beta3"
)

const (
	// These are internal finalizer values to Origin
	FinalizerOrigin kapi.FinalizerName = "openshift.io/origin"
)

// Address of a site
type SiteAddress struct {
	// URL to access the site
	Url string `json:"url"`
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
	StringSourceSpec
}

// StringSourceSpec specifies a string value, or external location
type StringSourceSpec struct {
	// Value specifies the cleartext value, or an encrypted value if keyFile is specified.
	Value string

	// Env specifies an envvar containing the cleartext value, or an encrypted value if the keyFile is specified.
	Env string

	// File references a file containing the cleartext value, or an encrypted value if a keyFile is specified.
	File string

	// KeyFile references a file containing the key to use to decrypt the value.
	KeyFile string
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
	Type SiteType `json:"type,omitempty"`
	// Address of the site
	Address SiteAddress `json:"address"`
	// The credential used to access site. It’s used for system routines (not behalf of users)
	Credential StringSource `json:"credential"`

	// Finalizers is an opaque list of values that must be empty to permanently remove object from storage
	Finalizers []kapi.FinalizerName
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

// Site metadata
type SiteMeta struct {
	// Version of the site
	Version string `json:"version,omitempty"`
}

// SiteStatus is information about the current status of a site.
type SiteStatus struct {
	// Phase is the recently observed lifecycle phase of the site.
	Phase            SitePhase `json:"phase,omitempty"`
	SiteMeta         string    `json:",inline"`
	SiteAgentAddress string    `json:"siteAgentAddress,omitempty"`
}

// Site information in Ubernetes
type Site struct {
	unversioned.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#metadata
	kapi.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the behavior of the Site.
	Spec SiteSpec `json:"spec,omitempty"`
	// Status describes the current status of a Site
	Status SiteStatus `json:"status,omitempty"`
}

// A list of Sites
type SiteList struct {
	unversioned.TypeMeta `json:",inline"`
	// Standard list metadata.
	// More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#types-kinds
	unversioned.ListMeta `json:"metadata,omitempty"`

	// List of Site objects.
	Items []Site `json:"items"`
}

// These constants represent annotations keys affixed to sites
const (
	// SiteDisplayName is an annotation that stores the name displayed when querying for sites
	SiteDisplayName = "openshift.io/display-name"
	// SiteDescription is an annotatoion that holds the description of the site
	SiteDescription = "openshift.io/description"
)
