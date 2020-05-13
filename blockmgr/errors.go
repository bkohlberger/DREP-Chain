package blockmgr

import (
	"errors"
)

var (
	ErrBlockNotFound         = errors.New("block not exist")
	ErrTxIndexOutOfRange     = errors.New("tx index out of range")
	ErrReachGasLimit         = errors.New("gas limit reached")
	ErrOverFlowMaxMsgSize    = errors.New("msg exceed max size")
	ErrEnoughPeer            = errors.New("peer exceed max peers")
	ErrNotContinueHeader     = errors.New("non contiguous header")
	ErrFindAncesstorTimeout  = errors.New("findAncestor timeout")
	ErrGetHeaderHashTimeout  = errors.New("get header hash timeout")
	ErrGetBlockTimeout       = errors.New("fetch blocks timeout")
	ErrReqStateTimeout       = errors.New("req state timeout")
	ErrDecodeMsg             = errors.New("fail to decode p2p msg")
	ErrMsgType               = errors.New("not expected msg type")
	ErrNegativeAmount        = errors.New("negative amount in tx")
	ErrExceedGasLimit        = errors.New("gas limit in tx has exceed block limit")
	ErrBalance               = errors.New("not enough balance")
	ErrNotSupportRenameAlias = errors.New("not suppport rename alias")
	ErrNoCommonAncesstor     = errors.New("no common ancesstor")
)
