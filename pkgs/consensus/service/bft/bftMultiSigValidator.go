package bft

import (
	"github.com/drep-project/binary"
	"github.com/drep-project/drep-chain/chain"
	"github.com/drep-project/drep-chain/crypto/secp256k1"
	"github.com/drep-project/drep-chain/crypto/secp256k1/schnorr"
	"github.com/drep-project/drep-chain/crypto/sha3"
	types2 "github.com/drep-project/drep-chain/pkgs/consensus/types"
	"github.com/drep-project/drep-chain/types"
)

type BlockMultiSigValidator struct {
	Producers types2.ProducerSet
}

func (blockMultiSigValidator *BlockMultiSigValidator) VerifyHeader(header, parent *types.BlockHeader) error {
	// check multisig
	// leader
	if !blockMultiSigValidator.Producers.IsLocalAddress(header.LeaderAddress) {
		return ErrBpNotInList
	}
	// minor
	for _, minor := range header.MinorAddresses {
		if !blockMultiSigValidator.Producers.IsLocalAddress(minor) {
			return ErrBpNotInList
		}
	}
	return nil
}

func (blockMultiSigValidator *BlockMultiSigValidator) VerifyBody(block *types.Block) error {
	participators := []*secp256k1.PublicKey{}
	multiSig := &MultiSignature{}
	err := binary.Unmarshal(block.Proof, multiSig)
	if err != nil {
		return err
	}
	for index, val := range multiSig.Bitmap {
		if val == 1 {
			producer := blockMultiSigValidator.Producers[index]
			participators = append(participators, producer.Pubkey)
		}
	}
	msg := block.AsSignMessage()
	sigmaPk := schnorr.CombinePubkeys(participators)

	if !schnorr.Verify(sigmaPk, sha3.Keccak256(msg), multiSig.Sig.R, multiSig.Sig.S) {
		return ErrMultiSig
	}
	return nil
}

func (blockMultiSigValidator *BlockMultiSigValidator) ExecuteBlock(context *chain.BlockExecuteContext) (types.Receipts, []*types.Log, uint64, error) {
	return nil, nil, 0, nil
}
