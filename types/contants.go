package types

import "math/big"

type TxType uint64

const (
	TransferType TxType = iota

	CreateContractType
	CallContractType

	SetAliasType          //给地址设置昵称
	VoteCreditType        //质押给别人
	CancelVoteCreditType  //撤销质押币
	CandidateType         //申请成为候选出块节点
	CancelCandidateType   //申请成为候选出块节点
	RegisterProducer
)

var (
	TransferGas         = big.NewInt(30000)
	MinerGas            = big.NewInt(20000)
	CreateContractGas   = big.NewInt(1000000)
	CallContractGas     = big.NewInt(10000000)
	CrossChainGas       = big.NewInt(10000000)
	SeAliasGas          = big.NewInt(10000000)
	RegisterProducerGas = big.NewInt(10000000)
)
