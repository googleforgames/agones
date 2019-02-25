package logfields

import (
	"github.com/sirupsen/logrus"
)

// ResourceType identifies the type of a resource for the purpose of putting it in structural logs.
type ResourceType string

// Identifiers used in logs for identifying Agones objects.
const (
	GameServerKey           ResourceType = "gsKey"
	GameServerSetKey        ResourceType = "gssKey"
	GameServerAllocationKey ResourceType = "gsaKey"
	FleetKey                ResourceType = "fleetKey"
	FleetAllocationKey      ResourceType = "faKey"
	FleetAutoscalerKey      ResourceType = "fasKey"
)

// AugmentLogEntry creates derived log entry with a given resource identifier ("namespace/name")
func AugmentLogEntry(base *logrus.Entry, resourceType ResourceType, resourceID string) *logrus.Entry {
	return base.WithField(string(resourceType), resourceID)
}
