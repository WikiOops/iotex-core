// Copyright (c) 2018 IoTeX
// This is an alpha (internal) release and is not suitable for production. This source code is provided 'as is' and no
// warranties are given as to title or non-infringement, merchantability or fitness for purpose and, to the extent
// permitted by law, all liability for your use of the code is disclaimed. This source code is governed by Apache
// License 2.0 that can be found in the LICENSE file.

package blocksync

import (
	"fmt"
	"sync"
	"syscall"
	"time"

	"github.com/iotexproject/iotex-core/actpool"
	bc "github.com/iotexproject/iotex-core/blockchain"
	cm "github.com/iotexproject/iotex-core/common"
	"github.com/iotexproject/iotex-core/common/routine"
	"github.com/iotexproject/iotex-core/config"
	"github.com/iotexproject/iotex-core/delegate"
	"github.com/iotexproject/iotex-core/logger"
	"github.com/iotexproject/iotex-core/network"
	pb "github.com/iotexproject/iotex-core/proto"
)

const (
	// Idle indicates an idle state
	Idle = iota
	// Init indicates the state when first block is received
	Init = Idle + 1
	// Active indicates the state after first block has been processed
	Active = Init + 1
)

// BlockSync defines the interface of blocksyncer
type BlockSync interface {
	Start() error
	Stop() error
	P2P() *network.Overlay
	ProcessSyncRequest(sender string, sync *pb.BlockSync) error
	ProcessBlock(blk *bc.Block) error
	ProcessBlockSync(blk *bc.Block) error
}

// blockSyncer implements BlockSync interface
type blockSyncer struct {
	mu             sync.RWMutex
	ackBlockCommit bool // acknowledges latest committed block
	ackBlockSync   bool // acknowledges old block from sync request
	ackSyncReq     bool // acknowledges incoming Sync request
	state          int
	syncHeight     uint64               // height of block at last sync request
	dropHeight     uint64               // height of most recent block being dropped
	currRcvdHeight uint64               // height of most recent incoming block
	lastRcvdHeight uint64               // height of last incoming block
	rcvdBlocks     map[uint64]*bc.Block // buffer of received blocks
	actionTime     time.Time
	sw             *SlidingWindow
	bc             bc.Blockchain
	ap             actpool.ActPool
	p2p            *network.Overlay
	task           *routine.RecurringTask
	fnd            string
	dp             delegate.Pool
}

// SyncTaskInterval returns the recurring sync task interval, or 0 if this config should not need to run sync task
func SyncTaskInterval(cfg *config.Config) time.Duration {
	if cfg.IsLightweight() {
		return time.Duration(0)
	}

	interval := cfg.BlockSync.Interval

	if cfg.IsFullnode() {
		// fullnode has less stringent requirement of staying in sync so can check less frequently
		interval <<= 2
	}
	return interval
}

// NewBlockSyncer returns a new block syncer instance
func NewBlockSyncer(cfg *config.Config, chain bc.Blockchain, ap actpool.ActPool, p2p *network.Overlay, dp delegate.Pool) BlockSync {
	sync := &blockSyncer{
		state:      Idle,
		rcvdBlocks: map[uint64]*bc.Block{},
		sw:         NewSlidingWindow(),
		bc:         chain,
		ap:         ap,
		p2p:        p2p,
		dp:         dp}

	sync.ackBlockCommit = cfg.IsDelegate() || cfg.IsFullnode()
	sync.ackBlockSync = cfg.IsDelegate() || cfg.IsFullnode()
	sync.ackSyncReq = cfg.IsDelegate() || cfg.IsFullnode()

	if interval := SyncTaskInterval(cfg); interval != 0 {
		sync.task = routine.NewRecurringTask(sync, interval)
	}

	delegates, err := dp.AllDelegates()
	if err != nil || len(delegates) == 0 {
		if err != nil {
			logger.Error().Err(err)
		} else {
			logger.Error().Msg("No delegates found")
		}
		syscall.Exit(syscall.SYS_EXIT)
	}

	switch cfg.NodeType {
	case config.DelegateType:
		// pick a delegate that is not myself
		if dlg := dp.AnotherDelegate(p2p.PRC.Addr); dlg != nil {
			sync.fnd = dlg.String()
		}
	case config.FullNodeType:
		// pick any valid delegate
		if dlg := dp.AnotherDelegate(""); dlg != nil {
			sync.fnd = dlg.String()
		}
	default:
		logger.Error().Str("node_type", cfg.NodeType).Msg("Unexpected node type.")
		return nil
	}
	return sync
}

// P2P returns the network overlay object
func (bs *blockSyncer) P2P() *network.Overlay {
	return bs.p2p
}

// Start starts a block syncer
func (bs *blockSyncer) Start() error {
	logger.Print("Starting block syncer")
	if bs.task != nil {
		bs.task.Init()
		bs.task.Start()
	}
	return nil
}

// Stop stops a block syncer
func (bs *blockSyncer) Stop() error {
	logger.Print("Stopping block syncer")
	if bs.task != nil {
		bs.task.Stop()
	}
	return nil
}

// Do checks the sliding window and send more sync request if needed
func (bs *blockSyncer) Do() {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	if bs.state == Idle {
		// simple exit if we haven't received any blocks
		return
	}
	if bs.state == Init {
		bs.processFirstBlock()
		bs.state = Active
		return
	}

	// This handles the case where a sync takes long time. By the time the window is closing, enough new
	// blocks are being dropped, so we check the window range and issue a new sync request
	if bs.state == Active && bs.sw.State != Open && bs.syncHeight < bs.dropHeight {
		bs.p2p.Tell(cm.NewTCPNode(bs.fnd), &pb.BlockSync{bs.syncHeight + 1, bs.dropHeight})
		logger.Warn().
			Str("addr", bs.p2p.PRC.Addr).
			Uint64("start", bs.syncHeight+1).
			Uint64("end", bs.dropHeight).
			Str("to", bs.fnd).
			Msg("+++++++++")
		if bs.dropHeight-bs.syncHeight > WindowSize {
			// trigger ProcessBlock() to drop incoming blocks, preventing too many blocks piling up in the buffer
			bs.sw.Update(bs.dropHeight)
			logger.Warn().
				Uint64("sync_height", bs.syncHeight+1).
				Uint64("drop_height", bs.dropHeight).
				Msg("++++reopen window")
		}
		bs.syncHeight = bs.dropHeight
		return
	}

	// health check if blocks keep coming in
	if bs.lastRcvdHeight == bs.currRcvdHeight {
		bs.state = Idle
		logger.Warn().
			Uint64("last_received_block", bs.lastRcvdHeight).
			Msg("No longer receiving blocks.")
	}
	bs.lastRcvdHeight = bs.currRcvdHeight
}

// ProcessSyncRequest processes a block sync request
func (bs *blockSyncer) ProcessSyncRequest(sender string, sync *pb.BlockSync) error {
	if !bs.ackSyncReq {
		// node is not meant to handle sync request, simply exit
		return nil
	}

	for i := sync.Start; i <= sync.End; i++ {
		blk, err := bs.bc.GetBlockByHeight(i)
		if err != nil {
			return err
		}
		// TODO: send back multiple blocks in one shot
		bs.p2p.Tell(cm.NewTCPNode(sender), &pb.BlockContainer{blk.ConvertToBlockPb()})
		//time.Sleep(time.Millisecond << 8)
	}
	return nil
}

// processFirstBlock processes an incoming latest committed block
func (bs *blockSyncer) processFirstBlock() error {
	height, err := bs.bc.TipHeight()
	if err != nil {
		return err
	}
	if bs.syncHeight = height; bs.currRcvdHeight > bs.syncHeight+1 {
		//TODO make it structured logging
		logger.Warn().Msgf(
			"++++++ [%s] Send first start = %d end = %d to %s",
			bs.p2p.PRC.Addr,
			bs.syncHeight+1,
			bs.currRcvdHeight,
			bs.fnd)
		bs.p2p.Tell(cm.NewTCPNode(bs.fnd), &pb.BlockSync{bs.syncHeight + 1, bs.currRcvdHeight})
	}
	if err := bs.sw.SetRange(bs.syncHeight, bs.currRcvdHeight); err != nil {
		return err
	}
	bs.syncHeight = bs.currRcvdHeight
	bs.actionTime = time.Now()
	return nil
}

// ProcessBlock processes an incoming latest committed block
func (bs *blockSyncer) ProcessBlock(blk *bc.Block) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	if !bs.ackBlockCommit {
		// node is not meant to handle latest committed block, simply exit
		return nil
	}
	height, err := bs.bc.TipHeight()
	if err != nil {
		return err
	}
	if bs.currRcvdHeight = blk.Height(); bs.currRcvdHeight <= height {
		err := fmt.Errorf(
			"****** [%s] Received block height %d <= Blockchain tip height %d",
			bs.p2p.PRC.Addr,
			bs.currRcvdHeight,
			height)
		return err
	}

	if bs.state == Idle && bs.currRcvdHeight == height+1 {
		// This is the special case where the first incoming block happens to be the next block following current
		// Blockchain tip, so we call ProcessFirstBlock() and let it proceed to commit right away
		// Otherwise waiting DO() to add it will cause thread context switch and incur extra latency, which usually
		// leads to a duplicate Consensus round on the same block height, increasing chance of Consensus failure
		if err := bs.processFirstBlock(); err != nil {
			return err
		}
		bs.state = Active
		//TODO make it structured logging
		logger.Warn().Msgf("====== receive tip block %d", bs.currRcvdHeight)
	}

	// TODO  Refancor the sync part to make logic clear and thorough
	// Just simply check the incoming blocks into the buffer
	//if bs.state == Idle || bs.state == Init {
	//	// indicate first valid block being received, DO() to handle sync request if needed
	//	bs.state = Init
	//	return nil
	//}

	if bs.state == Active && bs.sw.State == Open {
		// when window is open we are still WIP to sync old blocks, so simply drop incoming blocks
		bs.dropHeight = bs.currRcvdHeight
		//TODO make it structured logging
		logger.Warn().Msgf("****** [%s] drop block %d", bs.p2p.PRC.Addr, bs.currRcvdHeight)
		return nil
	}

	// check-in incoming block to the buffer
	if err := bs.checkBlockIntoBuffer(blk); err != nil {
		logger.Error().Err(err).Msg("")
		return nil
	}

	// commit all blocks in buffer that can be added to Blockchain
	return bs.commitBlocksInBuffer()
}

// ProcessBlockSync processes an incoming old block
func (bs *blockSyncer) ProcessBlockSync(blk *bc.Block) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	if !bs.ackBlockSync {
		// node is not meant to handle sync block, simply exit
		return nil
	}

	height, err := bs.bc.TipHeight()
	if err != nil {
		return err
	}
	if blk.Height() <= height {
		//TODO make it structured logging
		logger.Warn().Msgf(
			"****** [%s] Received block height %d <= Blockchain tip height %d",
			bs.p2p.PRC.Addr,
			blk.Height(),
			height)
		return nil
	}

	// check-in incoming block to the buffer
	bs.checkBlockIntoBuffer(blk)

	// commit all blocks in buffer that can be added to Blockchain
	return bs.commitBlocksInBuffer()
}

// checkBlockIntoBuffer adds a received blocks into the buffer
func (bs *blockSyncer) checkBlockIntoBuffer(blk *bc.Block) error {
	height := blk.Height()
	if bs.rcvdBlocks[height] != nil {
		return fmt.Errorf("|||||| [%s] discard existing block %d", bs.p2p.PRC.Addr, height)
	}
	bs.rcvdBlocks[height] = blk

	logger.Warn().
		Str("addr", bs.p2p.PRC.Addr).
		Uint64("block", height).
		Dur("interval", time.Since(bs.actionTime)).
		Msg("received block")
	bs.actionTime = time.Now()
	return nil
}

// commitBlocksInBuffer commits all blocks in the buffer that can be added to Blockchain
func (bs *blockSyncer) commitBlocksInBuffer() error {
	height, err := bs.bc.TipHeight()
	if err != nil {
		return err
	}
	next := height + 1
	for blk := bs.rcvdBlocks[next]; blk != nil; {
		if err := bs.bc.CommitBlock(blk); err != nil {
			return err
		}
		delete(bs.rcvdBlocks, next)

		// remove transfers in this block from ActPool and reset ActPool state
		bs.ap.Reset()

		//TODO make it structured logging
		logger.Warn().
			Str("name", bs.p2p.PRC.String()).
			Uint64("height", blk.Height()).
			Msg("commit a block")
		bs.actionTime = time.Now()

		// update sliding window
		bs.sw.Update(next)
		height, err = bs.bc.TipHeight()
		if err != nil {
			return err
		}
		next = height + 1
		blk = bs.rcvdBlocks[next]
	}
	return nil
}
