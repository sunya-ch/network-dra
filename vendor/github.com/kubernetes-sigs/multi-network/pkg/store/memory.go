package store

import (
	"sync"

	resourcev1beta1 "k8s.io/api/resource/v1beta1"
	"k8s.io/apimachinery/pkg/types"
)

type Memory struct {
	mu           sync.RWMutex
	podResources map[types.UID][]*resourcev1beta1.ResourceClaim
}

func NewMemory() *Memory {
	return &Memory{
		podResources: map[types.UID][]*resourcev1beta1.ResourceClaim{},
	}
}

func (m *Memory) Add(podUID types.UID, claim *resourcev1beta1.ResourceClaim) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.podResources[podUID] = append(m.podResources[podUID], claim)
}

func (m *Memory) Get(podUID types.UID) []*resourcev1beta1.ResourceClaim {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.podResources[podUID]
}

func (m *Memory) Delete(podUID types.UID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.podResources, podUID)
}
