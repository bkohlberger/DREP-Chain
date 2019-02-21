package service

import (
    "encoding/json"
    "github.com/AsynkronIT/protoactor-go/actor"
    "github.com/drep-project/drep-chain/app"
    chainTypes "github.com/drep-project/drep-chain/chain/types"
    "github.com/drep-project/drep-chain/common"
    "github.com/drep-project/drep-chain/crypto"
    "github.com/drep-project/drep-chain/crypto/secp256k1"
    "github.com/drep-project/drep-chain/crypto/sha3"
    "github.com/drep-project/drep-chain/database"
    "github.com/drep-project/drep-chain/log"
    p2pService "github.com/drep-project/drep-chain/network/service"
    p2pTypes "github.com/drep-project/drep-chain/network/types"
    rpcComponent "github.com/drep-project/drep-chain/rpc/component"
    rpcService "github.com/drep-project/drep-chain/rpc/service"
    "gopkg.in/urfave/cli.v1"
    "math/big"
    "strconv"
    "sync"
    "time"
)
var (
    rootChain common.ChainIdType
    genesisPubkey = "0x4f0160f718a479f9f14b3e70c10208d03839dec58380c24e0152d1cce44ce594"
)

type ChainService struct {
    RpcService       *rpcService.RpcService    `service:"rpc"`
    P2pServer       *p2pService.P2pService    `service:"p2p"`
    DatabaseService *database.DatabaseService `service:"database"`
    transactionPool *TransactionPool
    isRelay         bool
    apis            []app.API

    chainId common.ChainIdType

    lock sync.RWMutex
    addBlockSync sync.Mutex
    StartComplete  chan struct{}
    stopChanel   chan struct{}

    prvKey *secp256k1.PrivateKey

    CurrentHeight int64
    Config *chainTypes.ChainConfig
    pid *actor.PID
}


func (chainService *ChainService) Name() string {
    return "chain"
}

func (chainService *ChainService) Api() []app.API {
    return chainService.apis
}

func (chainService *ChainService) CommandFlags() ([]cli.Command, []cli.Flag)  {
    return nil, []cli.Flag{}
}

func (chainService *ChainService) P2pMessages() map[int]interface{} {
    return map[int]interface{}{
        chainTypes.MsgTypeBlockReq : chainTypes.BlockReq{},
        chainTypes.MsgTypeBlockResp : chainTypes.BlockResp{},
        chainTypes.MsgTypeBlock : chainTypes.Block{},
        chainTypes.MsgTypeTransaction : chainTypes.Transaction{},
        chainTypes.MsgTypePeerState : chainTypes.PeerState{},
        chainTypes.MsgTypeReqPeerState : chainTypes.ReqPeerState{},
    }
}

func (chainService *ChainService) Init(executeContext *app.ExecuteContext) error {
    chainService.Config = &chainTypes.ChainConfig{}
    err := executeContext.UnmashalConfig(chainService.Name(), chainService.Config )
    if err != nil {
        return err
    }

    chainService.CurrentHeight = chainService.DatabaseService.GetMaxHeight()
    if chainService.CurrentHeight == -1 {
        //generate genisis block
        genesisBlock := chainService.GenesisBlock()
        chainService.ExecuteTransactions(genesisBlock)
        chainService.CurrentHeight = chainService.DatabaseService.GetMaxHeight()
    }
    chainService.transactionPool = NewTransactionPool()
    props := actor.FromProducer(func() actor.Actor {
        return chainService
    })
    pid, err := actor.SpawnNamed(props, "chain_message")
    if err != nil {
        panic(err)
    }
    chainService.pid = pid
    router :=  chainService.P2pServer.Router
    chainP2pMessage := chainService.P2pMessages()
    for msgType, _ := range chainP2pMessage {
        router.RegisterMsgHandler(msgType,pid)
    }

    chainService.apis = []app.API{
        app.API{
            Namespace: "chain",
            Version:   "1.0",
            Service: &ChainApi{
                chain: chainService,
            },
            Public: true,
        },
    }
    return nil
}

func (chainService *ChainService) Start(executeContext *app.ExecuteContext) error {
    return nil
}

func (chainService *ChainService) Stop(executeContext *app.ExecuteContext) error {
    return nil
}

func (chainService *ChainService) sendBlock(block *chainTypes.Block) {
    chainService.P2pServer.Broadcast(block)
}

func (chainService *ChainService) ProcessBlock(block *chainTypes.Block) (*big.Int, error) {
    chainService.addBlockSync.Lock()
    defer chainService.addBlockSync.Unlock()
    log.Trace("Process block leader.", "LeaderPubKey", crypto.PubKey2Address(block.Header.LeaderPubKey).Hex(), " height ", strconv.FormatInt(block.Header.Height,10))
    return chainService.ExecuteTransactions(block)
}

func (chainService *ChainService) ProcessBlockReq(peer *p2pTypes.Peer, req *chainTypes.BlockReq) {
        from := req.Height + 1
        size := int64(200)
        for i := from; i <= chainService.DatabaseService.GetMaxHeight(); {
            bs := chainService.DatabaseService.GetBlocksFrom(i, size)
            resp := &chainTypes.BlockResp{Height : chainService.DatabaseService.GetMaxHeight(), Blocks : bs}
            chainService.P2pServer.Send(peer,resp)
            i += int64(len(bs))
        }

}

func (chainService *ChainService) GenerateBlock(leaderKey *secp256k1.PublicKey, members []*secp256k1.PublicKey) (*chainTypes.Block, error) {
    chainService.DatabaseService.BeginTransaction()
    defer  chainService.DatabaseService.Discard()

    height := chainService.DatabaseService.GetMaxHeight() + 1
    ts := chainService.transactionPool.PickTransactions(BlockGasLimit)

    gasUsed := new(big.Int)
    for _, t := range ts {
        g, _ := chainService.execute(t)
        gasUsed.Add(gasUsed, g)
    }

    timestamp := time.Now().Unix()
    previousHash := chainService.DatabaseService.GetPreviousBlockHash()

    stateRoot := chainService.DatabaseService.GetStateRoot()
    txHashes, _ := chainService.GetTxHashes(ts)
    merkle := chainService.DatabaseService.NewMerkle(txHashes)
    merkleRoot := merkle.Root.Hash

    var memberPks []*secp256k1.PublicKey
    for _, p := range members {
        memberPks = append(memberPks, p)
    }

    block := &chainTypes.Block{
        Header: &chainTypes.BlockHeader{
            Version:       Version,
            PreviousHash:  previousHash,
            ChainId:       chainService.chainId,
            GasLimit:      BlockGasLimit,
            GasUsed:       gasUsed,
            Timestamp:     timestamp,
            StateRoot:     stateRoot,
            MerkleRoot:    merkleRoot,
            TxHashes:      txHashes,
            Height:        height,
            LeaderPubKey : leaderKey,
            MinorPubKeys:  memberPks,
        },
        Data: &chainTypes.BlockData{
            TxCount: int32(len(ts)),
            TxList:  ts,
        },
    }
    return block, nil
}

func (chainService *ChainService) GetTxHashes(ts []*chainTypes.Transaction) ([][]byte, error) {
    txHashes := make([][]byte, len(ts))
    for i, tx := range ts {
        b, err := json.Marshal(tx.Data)
        if err != nil {
            return nil, err
        }
        txHashes[i] = sha3.Hash256(b)
    }
    return txHashes, nil
}

func (chainService *ChainService) Attach() (*rpcComponent.Client, error) {
    chainService.lock.RLock()
    defer chainService.lock.RUnlock()

    return rpcComponent.DialInProc(chainService.RpcService.IpcHandler), nil
}

func (chainService *ChainService) RootChain() common.ChainIdType {
    return rootChain
}

func (chainService *ChainService) GenerateBalanceTransaction(from *secp256k1.PublicKey, to crypto.CommonAddress, chainId common.ChainIdType, amount *big.Int) *chainTypes.Transaction {
    address := crypto.PubKey2Address(from)
    nonce := chainService.DatabaseService.GetNonce(address, chainId, false)
    data := &chainTypes.TransactionData{
        Version:   Version,
        Nonce:     nonce,
        Type:      TransferType,
        To:        to,
        ChainId:   chainId,
        Amount:    amount,
        GasPrice:  DefaultGasPrice,
        GasLimit:  TransferGas,
        Timestamp: time.Now().Unix(),
        PubKey:    from,
    }
    return &chainTypes.Transaction{Data: data}
}

func (chainService *ChainService) GenerateCreateContractTransaction(from *secp256k1.PublicKey, to crypto.CommonAddress, chainId common.ChainIdType, byteCode []byte) *chainTypes.Transaction {
    address := crypto.PubKey2Address(from)
    nonce := chainService.DatabaseService.GetNonce(address, chainId, false)
    nonce++
    data := &chainTypes.TransactionData{
        Nonce:     nonce,
        Type:      CreateContractType,
        ChainId:   chainId,
        GasPrice:  DefaultGasPrice,
        GasLimit:  CreateContractGas,
        Timestamp: time.Now().Unix(),
        Data:      make([]byte, len(byteCode) + 1),
        PubKey:    from,
    }
    copy(data.Data[1:], byteCode)
    data.Data[0] = 2
    return &chainTypes.Transaction{Data: data}
}

func (chainService *ChainService) GenerateCallContractTransaction(from *secp256k1.PublicKey, to crypto.CommonAddress, chainId common.ChainIdType, input []byte, amount *big.Int, readOnly bool) *chainTypes.Transaction {
    address := crypto.PubKey2Address(from)
    nonce := chainService.DatabaseService.GetNonce(address, chainId, false)
    nonce++
    data := &chainTypes.TransactionData{
        Nonce:     nonce,
        Type:      CallContractType,
        ChainId:   chainId,
        To:        to,
        Amount:    amount,
        GasPrice:  DefaultGasPrice,
        GasLimit:  CallContractGas,
        Timestamp: time.Now().Unix(),
        PubKey:    from,
        Data:      make([]byte, len(input) + 1),
    }
    copy(data.Data[1:], input)
    if readOnly {
        data.Data[0] = 1
    } else {
        data.Data[0] = 0
    }
    return &chainTypes.Transaction{Data: data}
}

func (chainService *ChainService) GenesisBlock() *chainTypes.Block{
    chainService.DatabaseService.BeginTransaction()
    defer chainService.DatabaseService.Discard()
    stateRoot := chainService.DatabaseService.GetStateRoot()
    merkle := chainService.DatabaseService.NewMerkle([][]byte{})
    merkleRoot := merkle.Root.Hash
    pubkey, _ := secp256k1.ParsePubKey([]byte(genesisPubkey))
    var memberPks []*secp256k1.PublicKey = nil
    return &chainTypes.Block{
        Header: &chainTypes.BlockHeader{
            Version: Version,
            PreviousHash: []byte{},
            GasLimit: BlockGasLimit,
            GasUsed: new(big.Int),
            Timestamp: 1545282765,
            StateRoot: stateRoot,
            MerkleRoot: merkleRoot,
            TxHashes: [][]byte{},
            Height: 0,
            LeaderPubKey : pubkey,
            MinorPubKeys : memberPks,
        },
        Data:&chainTypes.BlockData{
            TxCount:0,
            TxList:[]*chainTypes.Transaction{},
        },
    }
}