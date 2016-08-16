package api

import (
	"fmt"
)

const (
	displayNameOldAnnotation = "displayName"
	displayNameAnnotation    = "openshift.io/display-name"
)

// DisplayNameAndNameForSite returns a formatted string containing the name
// of the site and includes the display name if it differs.
func DisplayNameAndNameForSite(site *Site) string {
	displayName := site.Annotations[displayNameAnnotation]
	if len(displayName) == 0 {
		displayName = site.Annotations[displayNameOldAnnotation]
	}
	if len(displayName) > 0 && displayName != site.Name {
		return fmt.Sprintf("%s (%s)", displayName, site.Name)
	}
	return site.Name
}
