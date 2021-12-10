package vm

import (
	"time"

	"github.com/ava-labs/avalanchego/ids"

	"github.com/ava-labs/quarkvm/chain"
)

func (vm *VM) lookback(currTime int64, lastID ids.ID, f func(b *chain.StatelessBlock) (bool, error)) error {
	curr, err := vm.getBlock(lastID)
	if err != nil {
		return err
	}
	// Include at least parent block in the window, regardless of how old
	for curr != nil && (currTime-curr.Tmstmp <= chain.LookbackWindow || curr.ID() == lastID) {
		if cont, err := f(curr); !cont || err != nil {
			return err
		}
		if curr.Hght == 0 /* genesis */ {
			return nil
		}
		b, err := vm.getBlock(curr.Prnt)
		if err != nil {
			return err
		}
		curr = b
	}
	return nil
}

func (vm *VM) ValidBlockID(blockID ids.ID) (bool, error) {
	var foundBlockID bool
	err := vm.lookback(time.Now().Unix(), vm.preferred, func(b *chain.StatelessBlock) (bool, error) {
		if b.ID() == blockID {
			foundBlockID = true
			return false, nil
		}
		return true, nil
	})
	return foundBlockID, err
}

func (vm *VM) DifficultyEstimate() (uint64, error) {
	totalDifficulty := uint64(0)
	totalBlocks := uint64(0)
	err := vm.lookback(time.Now().Unix(), vm.preferred, func(b *chain.StatelessBlock) (bool, error) {
		totalDifficulty += b.Difficulty
		totalBlocks++
		return true, nil
	})
	return totalDifficulty/totalBlocks + 1, err
}
