package gameserversets

import (
	"sync"

	"agones.dev/agones/pkg/apis/stable/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// gameServerSetCacheEntry manages a list of items created and deleted locally for a single game server set.
type gameServerSetCacheEntry struct {
	mu              sync.Mutex
	pendingCreation map[string]*v1alpha1.GameServer
	pendingDeletion map[string]*v1alpha1.GameServer
}

func (e *gameServerSetCacheEntry) created(gs *v1alpha1.GameServer) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.pendingCreation == nil {
		e.pendingCreation = map[string]*v1alpha1.GameServer{}
	}
	e.pendingCreation[gs.Name] = gs.DeepCopy()
}

func (e *gameServerSetCacheEntry) deleted(gs *v1alpha1.GameServer) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.pendingDeletion == nil {
		e.pendingDeletion = map[string]*v1alpha1.GameServer{}
	}

	// Was pending creation, but deleted already.
	if _, ok := e.pendingCreation[gs.Name]; ok {
		delete(e.pendingCreation, gs.Name)
	}

	gsClone := gs.DeepCopy()
	t := metav1.Now()
	gsClone.ObjectMeta.DeletionTimestamp = &t
	e.pendingDeletion[gs.Name] = gsClone
}

// reconcileWithUpdatedServerList returns a list of game servers for a game server set taking into account
// the complete list of game servers passed as parameter and a list of pending creations and deletions.
func (e *gameServerSetCacheEntry) reconcileWithUpdatedServerList(list []*v1alpha1.GameServer) []*v1alpha1.GameServer {
	e.mu.Lock()
	defer e.mu.Unlock()

	var result []*v1alpha1.GameServer

	found := map[string]bool{}

	for _, gs := range list {
		if d := e.pendingDeletion[gs.Name]; d != nil {
			if !gs.ObjectMeta.DeletionTimestamp.IsZero() {
				// has deletion timestamp - return theirs
				result = append(result, d)
				delete(e.pendingDeletion, gs.Name)
			} else {
				result = append(result, d)
			}
		} else {
			// object not deleted locally, trust the list.
			result = append(result, gs)
		}

		if e.pendingCreation[gs.Name] != nil {
			// object was pending creation and now showed up in list results, remove local overlay.
			delete(e.pendingCreation, gs.Name)
		}

		found[gs.Name] = true
	}

	// now delete from 'pendingDeletion' all the items that were not found in the result.
	for gsName := range e.pendingDeletion {
		if !found[gsName] {
			// ("GSSC: %v is now fully deleted", gsName)
			delete(e.pendingDeletion, gsName)
		}
	}

	// add all game servers that are pending creation which were not in the list
	for _, gs := range e.pendingCreation {
		result = append(result, gs)
	}

	return result
}

// gameServerStateCache manages per-GSS cache of items created and deleted by this controller process
// to compensate for latency due to eventual consistency between client actions and K8s API server.
type gameServerStateCache struct {
	cache sync.Map
}

func (c *gameServerStateCache) forGameServerSet(gsSet *v1alpha1.GameServerSet) *gameServerSetCacheEntry {
	v, _ := c.cache.LoadOrStore(gsSet.ObjectMeta.Namespace+"/"+gsSet.ObjectMeta.Name, &gameServerSetCacheEntry{})
	return v.(*gameServerSetCacheEntry)
}

func (c *gameServerStateCache) deleteGameServerSet(gsSet *v1alpha1.GameServerSet) {
	c.cache.Delete(gsSet.ObjectMeta.Namespace + "/" + gsSet.ObjectMeta.Name)
}
