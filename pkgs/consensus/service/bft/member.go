package bft

import (
	"bytes"
	"errors"
	"github.com/drep-project/DREP-Chain/crypto/secp256k1"
	"github.com/drep-project/DREP-Chain/crypto/secp256k1/schnorr"
	"github.com/drep-project/DREP-Chain/crypto/sha3"
	consensusTypes "github.com/drep-project/DREP-Chain/pkgs/consensus/types"
	"github.com/drep-project/binary"
	"math/big"
	"sync"
	"time"
)

type Member struct {
	leader      *MemberInfo
	producers   []*MemberInfo
	liveMembers []*MemberInfo
	prvKey      *secp256k1.PrivateKey
	p2pServer   Sender

	msg     IConsenMsg
	msgHash []byte

	randomPrivakey *secp256k1.PrivateKey
	r              *big.Int

	waitTime time.Duration

	completed           chan struct{}
	timeOutChanel       chan struct{}
	errorChanel         chan error
	cancelWaitSetUp     chan struct{}
	cancelWaitChallenge chan struct{}
	currentState        int
	currentHeight       uint64
	stateLock           sync.RWMutex

	msgPool    chan *MsgWrap
	cancelPool chan struct{}
	validator  func(msg IConsenMsg) error
	convertor  func(msg []byte) (IConsenMsg, error)
}

func NewMember(prvKey *secp256k1.PrivateKey, p2pServer Sender, waitTime time.Duration, producers []*MemberInfo, minMember int, curHeight uint64, msgPool chan *MsgWrap) *Member {
	member := &Member{}
	member.prvKey = prvKey
	member.waitTime = waitTime
	member.p2pServer = p2pServer
	member.msgPool = msgPool
	member.producers = producers
	member.currentHeight = curHeight
	member.liveMembers = []*MemberInfo{}
	for _, producer := range producers {
		if producer.IsLeader {
			member.leader = producer
		} else {
			if producer.IsMe {
				//include self
				member.liveMembers = append(member.liveMembers, producer)
			} else {
				if producer.IsOnline {
					member.liveMembers = append(member.liveMembers, producer)
				}
			}
		}
	}
	member.cancelPool = make(chan struct{})
	member.Reset()
	return member
}

func (member *Member) Reset() {
	member.msg = nil
	member.msgHash = nil
	member.randomPrivakey = nil
	member.cancelPool = make(chan struct{})
	member.errorChanel = make(chan error, 1)
	member.completed = make(chan struct{}, 1)

	member.cancelWaitSetUp = make(chan struct{}, 1)
	member.timeOutChanel = make(chan struct{}, 1)
	member.cancelWaitChallenge = make(chan struct{}, 1)
	member.setState(INIT)
}

func (member *Member) ProcessConsensus(round int) (IConsenMsg, error) {
	defer func() {
		member.cancelPool <- struct{}{}
	}()

	log.WithField("Node", member.leader.Peer.IP()).Debug("wait for leader's setup message")
	member.setState(WAIT_SETUP)
	go member.WaitSetUp()
	go member.processP2pMessage(round)

	select {
	case err := <-member.errorChanel:
		log.WithField("Reason", err).Error("member consensus fail")
		return nil, err
	case <-member.timeOutChanel:
		member.setState(ERROR)
		return nil, ErrTimeout
	case <-member.completed:
		member.setState(COMPLETED)
		return member.msg, nil
	}

}
func (member *Member) processP2pMessage(round int) {

	for {
		select {
		case msg := <-member.msgPool:
			switch msg.Code {
			case MsgTypeSetUp:
				var setup Setup
				if err := binary.Unmarshal(msg.Msg, &setup); err != nil {
					log.Debugf("setup msg:%v err:%v", msg, err)
					continue
				}
				if setup.Round != round {
					log.WithField("come round", setup.Round).WithField("local round", round).Info("member process setup err")
					continue
				}
				go member.OnSetUp(msg.Peer, &setup)
			case MsgTypeChallenge:
				var challenge Challenge
				if err := binary.Unmarshal(msg.Msg, &challenge); err != nil {
					log.Debugf("challenge msg:%v err:%v", msg, err)
					continue
				}
				if challenge.Round != round {
					log.WithField("come round", challenge.Round).WithField("local round", round).Info("member process challege err")
					continue
				}
				go member.OnChallenge(msg.Peer, &challenge)
			case MsgTypeFail:
				var fail Fail
				if err := binary.Unmarshal(msg.Msg, &fail); err != nil {
					log.Debugf("challenge msg:%v err:%v", msg, err)
					continue
				}
				if fail.Round != round {
					log.WithField("come round", fail.Round).WithField("local round", round).Info("member process fail err")
					continue
				}
				go member.OnFail(msg.Peer, &fail)
			}
		case <-member.cancelPool:
			return
		}
	}
}
func (member *Member) WaitSetUp() {
	tm := time.NewTimer(member.waitTime)
	select {
	case <-tm.C:
		log.Debug("wait setup message timeout")
		member.setState(WAIT_SETUP_TIMEOUT)
		select {
		case member.timeOutChanel <- struct{}{}:
		default:
		}
		return
	case <-member.cancelWaitSetUp:
		return
	}
}

func (member *Member) OnSetUp(peer consensusTypes.IPeerInfo, setUp *Setup) {
	if member.currentHeight < setUp.Height {
		log.WithField("Receive Height", setUp.Height).
			WithField("Current Height", member.currentHeight).
			WithField("Status", member.getState()).
			Debug("setup low height")
		return
	} else if member.currentHeight > setUp.Height {
		log.WithField("Receive Height", setUp.Height).
			WithField("Current Height", member.currentHeight).
			WithField("Status", member.getState()).
			Debug("setup high height")
		return
	}

	if member.getState() != WAIT_SETUP {
		log.WithField("Receive Height", setUp.Height).
			WithField("Current Height", member.currentHeight).
			WithField("Status", member.getState()).
			Debug("setup error status")
		return
	}
	if member.leader.Peer.Equal(peer) {
		var err error
		member.msg, err = member.convertor(setUp.Msg)
		if err != nil || member.msg == nil {
			log.Errorf("convertor msg to block err:%s,height:%v,magic:0x%x,msg:%s", err.Error(), setUp.Height, setUp.Magic, setUp.String())
			return
		}

		member.msgHash = sha3.Keccak256(member.msg.AsSignMessage())
		member.commit(setUp.Round)
		log.Debug("sent commit message to leader")
		member.setState(WAIT_CHALLENGE)
		go member.WaitChallenge()
		select {
		case member.cancelWaitSetUp <- struct{}{}:
		default:
		}
	} else {
		log.Errorf("leader peer id:%s != peer:%s", member.leader.Peer.ID(), peer.ID())
		//check fail not response and start new round
		member.pushErrorMsg(ErrLeaderMistake)
	}
}

func (member *Member) WaitChallenge() {
	tm := time.NewTimer(member.waitTime)
	select {
	case <-tm.C:
		member.setState(WAIT_CHALLENGE_TIMEOUT)
		select {
		case member.timeOutChanel <- struct{}{}:
		default:
		}
		return
	case <-member.cancelWaitChallenge:
		return
	}
}

func (member *Member) OnChallenge(peer consensusTypes.IPeerInfo, challengeMsg *Challenge) {
	if member.currentHeight < challengeMsg.Height {
		log.WithField("Receive Height", challengeMsg.Height).
			WithField("Current Height", member.currentHeight).
			WithField("Status", member.getState()).
			Debug("challenge high height")
		return
	} else if member.currentHeight > challengeMsg.Height {
		log.WithField("Receive Height", challengeMsg.Height).
			WithField("Current Height", member.currentHeight).
			WithField("Status", member.getState()).
			Debug("challenge high height")
		return
	}
	if member.getState() != WAIT_CHALLENGE {
		log.WithField("Receive Height", challengeMsg.Height).
			WithField("Current Height", member.currentHeight).
			WithField("Status", member.getState()).
			Debug("challenge error status")
		return
	}
	log.Debug("recieved challenge message")
	if member.leader.Peer.Equal(peer) && bytes.Equal(member.msgHash, challengeMsg.R) {
		member.response(challengeMsg)
		log.Debug("response has sent")
		member.setState(COMPLETED)
		select {
		case member.cancelWaitChallenge <- struct{}{}:
		default:
		}
		member.completed <- struct{}{}
		return
	}
	member.pushErrorMsg(ErrChallenge)
	//check fail not response and start new round
}

func (member *Member) OnFail(peer consensusTypes.IPeerInfo, failMsg *Fail) {
	if member.currentHeight < failMsg.Height || member.getState() == COMPLETED || member.getState() == ERROR {
		return
	}
	log.WithField("msg", failMsg.Reason).Error("member receive leader's err message")
	member.pushErrorMsg(errors.New(failMsg.Reason))
}

func (member *Member) commit(round int) {
	if err := member.validator(member.msg); err != nil {
		log.WithField("Reason", err).Error("member check msg fail")
		member.pushErrorMsg(ErrValidateMsg)
		return
	}
	var err error
	var nouncePk *secp256k1.PublicKey

	member.randomPrivakey, nouncePk, err = schnorr.GenerateNoncePair(secp256k1.S256(), member.msgHash, member.prvKey, nil, schnorr.Sha256VersionStringRFC6979)
	if err != nil {
		member.pushErrorMsg(ErrGenerateNouncePriv)
		return
	}
	commitment := &Commitment{
		Round: round,
		Magic: CommitMagic,
		BpKey: member.prvKey.PubKey(),
		Q:     (*secp256k1.PublicKey)(nouncePk),
	}
	commitment.Height = member.currentHeight
	member.p2pServer.SendAsync(member.leader.Peer.GetMsgRW(), MsgTypeCommitment, commitment)
}

func (member *Member) response(challengeMsg *Challenge) {
	if bytes.Equal(member.msgHash, challengeMsg.R) {
		sig, err := schnorr.PartialSign(secp256k1.S256(), member.msgHash, member.prvKey, member.randomPrivakey, challengeMsg.SigmaQ)
		if err != nil {
			log.WithField("msg", err).Error("sign chanllenge error ")
			return
		}
		response := &Response{S: sig.Serialize()}
		response.BpKey = member.prvKey.PubKey()
		response.Height = member.currentHeight
		response.Magic = ResponseMagic
		response.Round = challengeMsg.Round
		member.p2pServer.SendAsync(member.leader.Peer.GetMsgRW(), MsgTypeResponse, response)
	} else {
		log.Error("commit messsage and chanllenge message not matched")
	}
}

/*
func (member *Member) getLiveMembers() []*consensusTypes.MemberInfo{
    return member.liveMembers
}
*/

func (member *Member) setState(state int) {
	member.stateLock.Lock()
	defer member.stateLock.Unlock()

	member.currentState = state
}

func (member *Member) getState() int {
	member.stateLock.RLock()
	defer member.stateLock.RUnlock()

	return member.currentState
}

func (member *Member) pushErrorMsg(msg error) {
	member.setState(ERROR)
CANCEL:
	for {
		select {
		case member.errorChanel <- msg:
		case member.cancelWaitSetUp <- struct{}{}:
		case member.cancelWaitChallenge <- struct{}{}:
		default:
			break CANCEL
		}
	}
}
