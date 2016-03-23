package site

import (
	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/rest"
	"k8s.io/kubernetes/pkg/watch"

	"github.com/openshift/origin/pkg/site/api"
)

// Registry is an interface for things that know how to store Sites.
type Registry interface {
	// ListSites obtains list of sites that match a selector.
	ListSites(ctx kapi.Context, options *kapi.ListOptions) (*api.SiteList, error)
	// GetSite retrieves a specific site.
	GetSite(ctx kapi.Context, siteID string) (*api.Site, error)
	// CreateSite creates a new site.
	CreateSite(ctx kapi.Context, site *api.Site) error
	// UpdateSite updates a site.
	UpdateSite(ctx kapi.Context, site *api.Site) error
	// DeleteSite deletes a site.
	DeleteSite(ctx kapi.Context, siteID string) error
	// WatchSites watches for new/modified/deleted sites.
	WatchSite(ctx kapi.Context, options *kapi.ListOptions) (watch.Interface, error)
}

// storage puts strong typing around storage calls

type storage struct {
	rest.StandardStorage
}

// NewRegistry returns a new Registry interface for the given Storage. Any mismatched
// types will panic.
func NewRegistry(s rest.StandardStorage) Registry {
	return &storage{s}
}

func (s *storage) ListSites(ctx kapi.Context, options *kapi.ListOptions) (*api.SiteList, error) {
	obj, err := s.List(ctx, options)
	if err != nil {
		return nil, err
	}
	return obj.(*api.SiteList), nil
}

func (s *storage) WatchSite(ctx kapi.Context, options *kapi.ListOptions) (watch.Interface, error) {
	return s.Watch(ctx, options)
}

func (s *storage) GetSite(ctx kapi.Context, siteID string) (*api.Site, error) {
	obj, err := s.Get(ctx, siteID)
	if err != nil {
		return nil, err
	}
	return obj.(*api.Site), nil
}

func (s *storage) CreateSite(ctx kapi.Context, site *api.Site) error {
	_, err := s.Create(ctx, site)
	return err
}

func (s *storage) UpdateSite(ctx kapi.Context, site *api.Site) error {
	_, _, err := s.Update(ctx, site)
	return err
}

func (s *storage) DeleteSite(ctx kapi.Context, siteID string) error {
	_, err := s.Delete(ctx, siteID, nil)
	return err
}
