package protocol

import (
	"testing"
)

func TestNextLeaderTime(t *testing.T) {
	cases := []struct {
		desc               string
		startBlockHeight   uint64
		bestBlockHeight    uint64
		startTime          uint64
		nodeOrder          uint64
		wantError          error
		wantNextLeaderTime int64
	}{
		{
			desc:               "normal case",
			startBlockHeight:   1000,
			bestBlockHeight:    1500,
			startTime:          1557906284061,
			nodeOrder:          1,
			wantError:          nil,
			wantNextLeaderTime: 1557906537561,
		},
		{
			desc:               "best block height equals to start block height",
			startBlockHeight:   1000,
			bestBlockHeight:    1000,
			startTime:          1557906284061,
			nodeOrder:          0,
			wantError:          nil,
			wantNextLeaderTime: 1557906284061,
		},
		{
			desc:               "best block height equals to start block height",
			startBlockHeight:   1000,
			bestBlockHeight:    1000,
			startTime:          1557906284061,
			nodeOrder:          1,
			wantError:          nil,
			wantNextLeaderTime: 1557906284061 + blockNumEachNode * blockTimeInterval,
		},
		{
			desc:               "has no chance product block in this round of voting",
			startBlockHeight:   1000,
			bestBlockHeight:    1995,
			startTime:          1557906284061,
			nodeOrder:          1,
			wantError:          errHasNoChanceProductBlock,
			wantNextLeaderTime: 0,
		},
		{
			desc:               "the node is producting block",
			startBlockHeight:   1000,
			bestBlockHeight:    1001,
			startTime:          1557906284061,
			nodeOrder:          0,
			wantError:          nil,
			wantNextLeaderTime: 1557906284061,
		},
		{
			desc:               "the node is producting block",
			startBlockHeight:   1000,
			bestBlockHeight:    1067,
			startTime:          1557906284061,
			nodeOrder:          1,
			wantError:          nil,
			wantNextLeaderTime: 1557906284061 + 66 * blockTimeInterval,
		},
		{
			desc:               "first round, must exclude genesis block",
			startBlockHeight:   1,
			bestBlockHeight:    5,
			startTime:          1557906284061,
			nodeOrder:          3,
			wantError:          nil,
			wantNextLeaderTime: 1557906284061 + 9 * blockTimeInterval,
		},
	}

	for i, c := range cases {
		nextLeaderTime, err := nextLeaderTimeHelper(c.startBlockHeight, c.bestBlockHeight, c.startTime, c.nodeOrder)
		if err != c.wantError {
			t.Fatalf("case #%d (%s) want error:%v, got error:%v", i, c.desc, c.wantError, err)
		}

		if err != nil {
			continue
		}
		nextLeaderTimestamp := nextLeaderTime.UnixNano() / 1e6
		if nextLeaderTimestamp != c.wantNextLeaderTime {
			t.Errorf("case #%d (%s) want next leader time:%d, got next leader time:%d", i, c.desc, c.wantNextLeaderTime, nextLeaderTimestamp)
		}
	}
}
