package mov

import (
	"runtime"
	"sync"

	"github.com/bytom/vapor/application/mov/common"
	"github.com/bytom/vapor/application/mov/database"
	"github.com/bytom/vapor/application/mov/match"
	"github.com/bytom/vapor/protocol/bc/types"
)

type matchCollector struct {
	engine            *match.Engine
	tradePairIterator *database.TradePairIterator
	gasLeft           int64
	isTimeout         func() bool

	workerNum   int
	workerNumCh chan int
	processCh   chan *matchTxResult
	tradePairCh chan *common.TradePair
	closeCh     chan struct{}
}

type matchTxResult struct {
	matchedTx *types.Tx
	err       error
}

func newMatchTxCollector(engine *match.Engine, iterator *database.TradePairIterator, gasLeft int64, isTimeout func() bool) *matchCollector {
	workerNum := runtime.NumCPU()
	return &matchCollector{
		engine:            engine,
		tradePairIterator: iterator,
		workerNum:         workerNum,
		workerNumCh:       make(chan int, workerNum),
		processCh:         make(chan *matchTxResult),
		tradePairCh:       make(chan *common.TradePair, workerNum),
		closeCh:           make(chan struct{}),
		gasLeft:           gasLeft,
		isTimeout:         isTimeout,
	}
}

func (m *matchCollector) result() ([]*types.Tx, error) {
	var wg sync.WaitGroup
	for i := 0; i < int(m.workerNum); i++ {
		wg.Add(1)
		go m.matchTxWorker(&wg)
	}

	wg.Add(1)
	go m.tradePairProducer(&wg)

	matchedTxs, err := m.collect()
	// wait for all goroutine release
	wg.Wait()
	return matchedTxs, err
}

func (m *matchCollector) collect() ([]*types.Tx, error) {
	defer close(m.closeCh)

	var matchedTxs []*types.Tx
	completed := 0
	for !m.isTimeout() {
		select {
		case data := <-m.processCh:
			if data.err != nil {
				return nil, data.err
			}

			gasUsed := calcMatchedTxGasUsed(data.matchedTx)
			if m.gasLeft -= gasUsed; m.gasLeft >= 0 {
				matchedTxs = append(matchedTxs, data.matchedTx)
			} else {
				return matchedTxs, nil
			}
		case <-m.workerNumCh:
			if completed++; completed == m.workerNum {
				return matchedTxs, nil
			}
		}
	}
	return matchedTxs, nil
}

func (m *matchCollector) tradePairProducer(wg *sync.WaitGroup) {
	defer func() {
		close(m.tradePairCh)
		wg.Done()
	}()

	tradePairMap := make(map[string]bool)

	for m.tradePairIterator.HasNext() {
		tradePair := m.tradePairIterator.Next()
		if tradePairMap[tradePair.Key()] {
			continue
		}

		tradePairMap[tradePair.Key()] = true
		tradePairMap[tradePair.Reverse().Key()] = true

		select {
		case <-m.closeCh:
			return
		case m.tradePairCh <- tradePair:
		}
	}
}

func (m *matchCollector) matchTxWorker(wg *sync.WaitGroup) {
	defer func() {
		m.workerNumCh <- 1
		wg.Done()
	}()

	for {
		select {
		case <-m.closeCh:
			return
		case tradePair := <-m.tradePairCh:
			if tradePair == nil {
				return
			}
			for m.engine.HasMatchedTx(tradePair, tradePair.Reverse()) {
				matchedTx, err := m.engine.NextMatchedTx(tradePair, tradePair.Reverse())
				data := &matchTxResult{matchedTx: matchedTx, err: err}
				select {
				case <-m.closeCh:
					return
				case m.processCh <- data:
					if data.err != nil {
						return
					}
				}
			}
		}

	}
}
