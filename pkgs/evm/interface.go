package evm

import (
	"github.com/drep-project/drep-chain/app"
	"github.com/drep-project/drep-chain/pkgs/evm/vm"
	"github.com/drep-project/drep-chain/types"
	"math/big"
)

type Vm interface {
	app.Service
	Eval(vm.VMState, *types.Transaction, *types.BlockHeader, ChainContext, uint64, *big.Int) (ret []byte, gasUsed uint64, failed bool, err error)
}
