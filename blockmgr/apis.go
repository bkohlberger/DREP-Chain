package blockmgr

import (
	"math/big"

	"github.com/drep-project/DREP-Chain/common"
	"github.com/drep-project/DREP-Chain/crypto"
	"github.com/drep-project/DREP-Chain/database"
	"github.com/drep-project/DREP-Chain/types"
	"github.com/drep-project/binary"
)

/*
name: 区块
usage: 用于处理区块链偏上层逻辑
prefix:blockMgr
*/
type BlockMgrApi struct {
	blockMgr  *BlockMgr
	dbService *database.DatabaseService
}

/*
 name: sendRawTransaction
 usage: 发送已签名的交易.
 params:
	1. 已签名的交易
 return: 交易hash
 example: curl http://localhost:15645 -X POST --data '{"jsonrpc":"2.0","method":"blockmgr_sendRawTransaction","params":["0x40a287b6d30b05313131317a4120dd8c23c40910d038fa43b2f8932d3681cbe5ee3079b6e9de0bea6e8e6b2a867a561aa26e1cd6b62aa0422a043186b593b784bf80845c3fd5a7fbfe62e61d8564"], "id": 3}' -H "Content-Type:application/json"
 response:
	{"jsonrpc":"2.0","id":1,"result":"0xf30e858667fa63bc57ae395c3f57ede9bb3ad4969d12f4bce51d900fb5931538"}
*/
func (blockMgrApi *BlockMgrApi) SendRawTransaction(txbytes common.Bytes) (string, error) {

	tx := &types.Transaction{}
	err := binary.Unmarshal(txbytes, tx)
	if err != nil {
		return "", err
	}
	err = blockMgrApi.blockMgr.SendTransaction(tx, true)
	if err != nil {
		return "", err
	}
	blockMgrApi.blockMgr.BroadcastTx(types.MsgTypeTransaction, tx, true)
	return tx.TxHash().String(), err
}

/*
 name: gasPrice
 usage: 获取系统的给出的gasprice建议值
 params:
	1. 待查询地址
 return: 价格和是否错误信息
 example: curl http://localhost:15645 -X POST --data '{"jsonrpc":"2.0","method":"blockmgr_gasPrice","params":[], "id": 3}' -H "Content-Type:application/json"
 response:
*/
func (blockMgrApi *BlockMgrApi) GasPrice() (*big.Int, error) {
	return blockMgrApi.blockMgr.gpo.SuggestPrice()
}

/*
 name: GetPoolTransactions
 usage: 获取交易池中的交易信息.
 params:
	1. 待查询地址
 return: 交易池中所有交易
 example: curl http://localhost:15645 -X POST --data '{"jsonrpc":"2.0","method":"blockmgr_getPoolTransactions","params":["0x8a8e541ddd1272d53729164c70197221a3c27486"], "id": 3}' -H "Content-Type:application/json"
 response:
*/
func (blockMgrApi *BlockMgrApi) GetPoolTransactions(addr *crypto.CommonAddress) []types.Transactions {
	return blockMgrApi.blockMgr.GetPoolTransactions(addr)
}

/*
 name: GetTransactionCount
 usage: 获取地址发出的交易总数
 params:
	1. 待查询地址
 return: 交易池中所有交易
 example: curl http://localhost:15645 -X POST --data '{"jsonrpc":"2.0","method":"blockmgr_getTransactionCount","params":["0x8a8e541ddd1272d53729164c70197221a3c27486"], "id": 3}' -H "Content-Type:application/json"
 response:
*/
func (blockMgrApi *BlockMgrApi) GetTransactionCount(addr *crypto.CommonAddress) uint64 {
	return blockMgrApi.blockMgr.GetTransactionCount(addr)
}

/*
 name: GetPoolMiniPendingNonce
 usage: 获取pending队列中，最小的Nonce
 params:
	1. 待查询地址
 return: pending 队列中最小的nonce
 example: curl http://localhost:15645 -X POST --data '{"jsonrpc":"2.0","method":"blockmgr_getPoolMiniPendingNonce","params":["0x8a8e541ddd1272d53729164c70197221a3c27486"], "id": 3}' -H "Content-Type:application/json"
 response:
*/

func (blockMgrApi *BlockMgrApi) GetPoolMiniPendingNonce(addr *crypto.CommonAddress) uint64 {
	return blockMgrApi.blockMgr.GetPoolMiniPendingNonce(addr)
}

/*
 name: GetTxInPool
 usage: 查询交易是否在交易池，如果在，返回交易
 params:
	1. 发起转账的地址

 return: 交易完整信息
 example: curl -H "Content-Type: application/json" -X post --data '{"jsonrpc":"2.0","method":"blockmgr_getTxInPool","params":["0x3ebcbe7cb440dd8c52940a2963472380afbb56c5"],"id":1}' http://127.0.0.1:15645
 response:
   {
  "jsonrpc": "2.0",
  "id": 3,
  "result": {
    "Hash": "0xfa5c34114ff459b4c97e7cd268c507c0ccfcfc89d3ccdcf71e96402f9899d040",
    "From": "0x7923a30bbfbcb998a6534d56b313e68c8e0c594a",
    "Version": 1,
    "Nonce": 15632,
    "Type": 0,
    "To": "0x7923a30bbfbcb998a6534d56b313e68c8e0c594a",
    "ChainId": "00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
    "Amount": "0x111",
    "GasPrice": "0x110",
    "GasLimit": "0x30000",
    "Timestamp": 1559322808,
    "Data": null,
    "Sig": "0x20f25b86c4bf73aa4fa0bcb01e2f5731de3a3917c8861d1ce0574a8d8331aedcf001e678000f6afc95d35a53ef623a2055fce687f85c2fd752dc455ab6db802b1f"
  }
}
*/
func (blockMgrApi *BlockMgrApi) GetTxInPool(hash string) (*types.Transaction, error) {
	tx, err := blockMgrApi.blockMgr.transactionPool.GetTxInPool(hash)
	if err != nil {
		return nil, err
	}
	return tx, nil
}
