package component

import (
	"encoding/json"
	"github.com/drep-project/DREP-Chain/crypto"
	"github.com/drep-project/DREP-Chain/types"
	"sync"
)

type MemoryStore struct {
	keys *sync.Map
}

func (mstore *MemoryStore) ExportAddrs(auth string) ([]string, error) {
	addrs := make([]string, 0)
	mstore.keys.Range(func(key, value interface{}) bool {
		addrs = append(addrs, value.(types.Node).Address.String())
		return true
	})
	return addrs, nil
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{keys: new(sync.Map)}
}

func (mstore *MemoryStore) GetKey(addr *crypto.CommonAddress, auth string) (*types.Node, error) {
	node, ok := mstore.keys.Load(addr.String())
	if ok {
		return copy(node.(*types.Node)), nil
	} else {
		return nil, ErrKeyNotFound
	}

}

func (mstore *MemoryStore) StoreKey(k *types.Node, auth string) error {
	k = copy(k)
	mstore.keys.Store(k.Address.String(), k)
	return nil
}

func (mstore *MemoryStore) ExportKey(auth string) ([]*types.Node, error) {
	nodes := []*types.Node{}
	mstore.keys.Range(func(key, value interface{}) bool {
		nodes = append(nodes, copy(value.(*types.Node)))
		return true
	})
	return nodes, nil
}

func (mstore *MemoryStore) JoinPath(filename string) string {
	return ""
}

func copy(node *types.Node) *types.Node {
	bytes, _ := json.Marshal(node)
	newNode := &types.Node{}
	json.Unmarshal(bytes, newNode)
	return newNode
}
