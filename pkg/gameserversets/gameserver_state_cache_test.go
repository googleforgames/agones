package gameserversets

import (
	"sort"
	"testing"

	"agones.dev/agones/pkg/apis/stable/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var deletionTime = metav1.Now()

func TestGameServerStateCache(t *testing.T) {
	var cache gameServerStateCache
	gsSet1 := &v1alpha1.GameServerSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "set-1",
			Namespace: "ns1",
		},
	}
	gsSet1b := &v1alpha1.GameServerSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "set-1",
			Namespace: "ns2",
		},
	}
	gsSet2 := &v1alpha1.GameServerSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "set-2",
			Namespace: "ns1",
		},
	}

	entry1 := cache.forGameServerSet(gsSet1)
	if got, want := entry1, cache.forGameServerSet(gsSet1); got != want {
		t.Errorf("unexpectedly different entries for the same set")
	}
	if got, want := entry1, cache.forGameServerSet(gsSet1b); got == want {
		t.Errorf("unexpectedly same entries for different sets 1 and 1b")
	}
	if got, want := entry1, cache.forGameServerSet(gsSet2); got == want {
		t.Errorf("unexpectedly same entries for different sets 1 and 2")
	}
	if got, want := entry1, cache.forGameServerSet(gsSet1); got != want {
		t.Errorf("unexpectedly different entries for the same set")
	}
	cache.deleteGameServerSet(gsSet1)
	if got, want := entry1, cache.forGameServerSet(gsSet1); got == want {
		t.Errorf("unexpectedly same entries for sets 1  before and after deletion")
	}
}

func TestGameServerSetCacheEntry(t *testing.T) {
	gs1 := makeGameServer("gs-1")
	gs2 := makeGameServer("gs-2")

	cases := []struct {
		desc                     string
		setup                    func(c *gameServerSetCacheEntry)
		list                     []*v1alpha1.GameServer
		expected                 []*v1alpha1.GameServer
		expectedPendingCreations int
		expectedPendingDeletions int
	}{
		{
			desc:     "EmptyList",
			setup:    func(c *gameServerSetCacheEntry) {},
			list:     nil,
			expected: nil,
		},
		{
			desc:                     "LocallyAddedGameServerNotInServerResults",
			setup:                    func(c *gameServerSetCacheEntry) { c.created(gs1) },
			list:                     nil,
			expected:                 []*v1alpha1.GameServer{gs1},
			expectedPendingCreations: 1,
		},
		{
			desc:                     "LocallyAddedGameServerAnotherOneFoundOnServer",
			setup:                    func(c *gameServerSetCacheEntry) { c.created(gs1) },
			list:                     []*v1alpha1.GameServer{gs2},
			expected:                 []*v1alpha1.GameServer{gs1, gs2},
			expectedPendingCreations: 1,
		},
		{
			desc:     "LocallyAddedGameServerAlsoFoundOnServer",
			setup:    func(c *gameServerSetCacheEntry) { c.created(gs1) },
			list:     []*v1alpha1.GameServer{gs1},
			expected: []*v1alpha1.GameServer{gs1},
		},
		{
			desc:                     "LocallyDeletedStillFoundOnServer",
			setup:                    func(c *gameServerSetCacheEntry) { c.deleted(gs1) },
			list:                     []*v1alpha1.GameServer{gs1},
			expected:                 []*v1alpha1.GameServer{deleted(gs1)},
			expectedPendingDeletions: 1,
		},
		{
			desc:     "LocallyDeletedNotFoundOnServer",
			setup:    func(c *gameServerSetCacheEntry) { c.deleted(gs1) },
			list:     []*v1alpha1.GameServer{},
			expected: []*v1alpha1.GameServer{},
		},
		{
			desc: "LocallyCreatedAndDeletedFoundOnServer",
			setup: func(c *gameServerSetCacheEntry) {
				c.created(gs1)
				c.deleted(gs1)
			},
			list:                     []*v1alpha1.GameServer{gs1},
			expected:                 []*v1alpha1.GameServer{deleted(gs1)},
			expectedPendingDeletions: 1,
		},
		{
			desc: "LocallyCreatedAndDeletedFoundDeletedOnServer",
			setup: func(c *gameServerSetCacheEntry) {
				c.created(gs1)
				c.deleted(gs1)
			},
			list:     []*v1alpha1.GameServer{deleted(gs1)},
			expected: []*v1alpha1.GameServer{deleted(gs1)},
		},
		{
			desc: "LocallyCreatedAndDeletedNotFoundOnServer",
			setup: func(c *gameServerSetCacheEntry) {
				c.created(gs1)
				c.deleted(gs1)
			},
			list:     []*v1alpha1.GameServer{},
			expected: []*v1alpha1.GameServer{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			entry := &gameServerSetCacheEntry{}

			tc.setup(entry)
			result := entry.reconcileWithUpdatedServerList(tc.list)
			assert.Equal(t, sortedStatusInfo(result), sortedStatusInfo(tc.expected))

			if got, want := len(entry.pendingCreation), tc.expectedPendingCreations; got != want {
				t.Errorf("unexpected # of pending creations %v, wanted %v", got, want)
			}
			if got, want := len(entry.pendingDeletion), tc.expectedPendingDeletions; got != want {
				t.Errorf("unexpected # of pending deletions %v, wanted %v", got, want)
			}

			result2 := entry.reconcileWithUpdatedServerList(result)
			assert.Equal(t, sortedStatusInfo(result2), sortedStatusInfo(result))

			// now both pending creations and deletions must be zero
			if got, want := len(entry.pendingCreation), 0; got != want {
				t.Errorf("unexpected # of pending creations %v, wanted %v", got, want)
			}
			if got, want := len(entry.pendingDeletion), 0; got != want {
				t.Errorf("unexpected # of pending deletions %v, wanted %v", got, want)
			}
		})
	}
}

func makeGameServer(s string) *v1alpha1.GameServer {
	return &v1alpha1.GameServer{
		ObjectMeta: metav1.ObjectMeta{
			Name: s,
		},
	}
}

func deleted(gs *v1alpha1.GameServer) *v1alpha1.GameServer {
	gs2 := gs.DeepCopy()
	gs2.ObjectMeta.DeletionTimestamp = &deletionTime
	return gs2
}

type gameServerStatusInfo struct {
	name    string
	status  string
	deleted bool
}

func sortedStatusInfo(list []*v1alpha1.GameServer) []gameServerStatusInfo {
	var result []gameServerStatusInfo
	for _, gs := range list {
		result = append(result, gameServerStatusInfo{
			gs.Name,
			string(gs.Status.State),
			gs.DeletionTimestamp != nil,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].name < result[j].name
	})
	return result
}
