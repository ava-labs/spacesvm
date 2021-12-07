package chain

import (
	"time"

	"github.com/ava-labs/avalanchego/database/versiondb"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow/consensus/snowman"
	log "github.com/inconshreveable/log15"
)

func BuildBlock(vm VM, preferred ids.ID) (snowman.Block, error) {
	log.Debug("attempting block building")

	nextTime := time.Now().Unix()
	prnt, err := vm.GetBlock(preferred)
	if err != nil {
		log.Debug("block building failed: couldn't get parent", "err", err)
		return nil, err
	}
	parent := prnt.(*Block)
	context, err := vm.ExecutionContext(nextTime, parent)
	if err != nil {
		log.Debug("block building failed: couldn't get execution context", "err", err)
		return nil, err
	}
	b := NewBlock(vm, parent, nextTime, context)

	// Select new transactions
	parentDB, err := parent.onAccept()
	if err != nil {
		log.Debug("block building failed: couldn't get parent db", "err", err)
		return nil, err
	}
	vdb := versiondb.New(parentDB)
	b.Txs = []*Transaction{}
	vm.MempoolPrune(context.RecentBlockIDs) // clean out invalid txs
	for len(b.Txs) < TargetTransactions && vm.MempoolSize() > 0 {
		next, diff := vm.MempoolNext()
		if diff < b.Difficulty {
			vm.MempoolPush(next)
			log.Debug("skipping tx: too low difficulty", "block diff", b.Difficulty, "tx diff", next.Difficulty())
			break
		}
		// Verify that changes pass
		tvdb := versiondb.New(vdb)
		if err := next.Verify(tvdb, b.Tmstmp, context); err != nil {
			log.Debug("skipping tx: failed verification", "err", err)
			continue
		}
		if err := tvdb.Commit(); err != nil {
			return nil, err
		}
		// Wait to add prefix until after verification
		b.Txs = append(b.Txs, next)
	}

	// Compute block hash and marshaled representation
	if err := b.init(); err != nil {
		return nil, err
	}

	// Verify block to ensure it is formed correctly
	_, vdb, err = b.verify()
	if err != nil {
		log.Debug("block building failed: failed verification", "err", err)
		return nil, err
	}
	return b, nil
}
