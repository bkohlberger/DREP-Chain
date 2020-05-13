package bft

import "github.com/drep-project/DREP-Chain/pkgs/consensus/types"

type MemberInfo struct {
	Peer     types.IPeerInfo
	Producer *Producer
	Status   int
	IsMe     bool
	IsLeader bool
	IsOnline bool
}
