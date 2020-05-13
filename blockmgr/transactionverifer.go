package blockmgr

import (
	"github.com/drep-project/DREP-Chain/chain"
	"github.com/drep-project/DREP-Chain/types"
)

// VerifyTransaction use current tip state as environment may not matched read disk state
// not check tx nonce ; current nonce shoud use pool nonce while receive tx
func (blockMgr *BlockMgr) verifyTransaction(tx *types.Transaction) error {
	//db := blockMgr.ChainService.GetCurrentState()
	//from, err := tx.From()

	// Transactions can't be negative. This may never happen using RLP decoded
	// transactions but may occur if you create a transaction using the RPC.
	if tx.Amount().Sign() < 0 {
		return ErrNegativeAmount
	}

	tip := blockMgr.ChainService.BestChain().Tip()
	// Check the transaction doesn't exceed the current
	// block limit gas.
	if tip.GasLimit.Uint64() < tx.Gas() {
		return ErrExceedGasLimit
	}

	// Transactor should have enough funds to cover the costs
	// cost == V + GP * GL
	//originBalance := db.GetBalance(from)
	//if originBalance.Cmp(tx.Cost()) < 0 {
	//	return ErrBalance
	//}

	// Should supply enough intrinsic gas
	gas, err := tx.IntrinsicGas()
	if err != nil {
		return err
	}
	if tx.Gas() < gas {
		log.WithField("gas", gas).WithField("tx.gas", tx.Gas()).Error("gas exceed tx gaslimit ")
		return ErrReachGasLimit
	}
	if tx.Type() == types.SetAliasType {
		from, err := tx.From()
		if err != nil {
			return err
		}
		newAlias := tx.GetData()
		if newAlias == nil {
			return chain.ErrUnsupportAliasChar
		}
		if err := chain.CheckAlias(newAlias); err != nil {
			return err
		}
		trieQuery, err := chain.NewTrieQuery(blockMgr.DatabaseService.LevelDb(), tip.StateRoot)
		if err != nil {
			return err
		}
		alias := trieQuery.GetStorageAlias(from)
		if alias != "" {
			return ErrNotSupportRenameAlias
		}
	}
	return nil
}
