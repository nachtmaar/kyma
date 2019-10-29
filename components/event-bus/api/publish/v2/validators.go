package v2

import (
	"regexp"

	api "github.com/kyma-project/kyma/components/event-bus/api/publish"
)

var (
	// channel name components
	// TODO(nachtmaar): only used by tests
	isValidSourceID         = regexp.MustCompile(api.AllowedSourceIDChars).MatchString
	isValidEventType        = regexp.MustCompile(api.AllowedEventTypeChars).MatchString
	isValidEventTypeVersion = regexp.MustCompile(api.AllowedEventTypeVersionChars).MatchString
)
