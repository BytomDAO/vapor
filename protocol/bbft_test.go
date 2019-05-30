package protocol

import (
	"testing"
)

func TestNextLeaderTime(t *testing.T) {
	cases := []struct {
		desc               string
		startTime          uint64
		now                uint64
		nodeOrder          uint64
		wantError          error
		wantNextLeaderTime uint64
	}{
		{
			desc:               "normal case",
			startTime:          1557906284061,
			now:                1557906534061,
			nodeOrder:          1,
			wantError:          nil,
			wantNextLeaderTime: 1557906537561,
		},
		{
			desc:               "best block height equals to start block height",
			startTime:          1557906284061,
			now:                1557906284061,
			nodeOrder:          0,
			wantError:          nil,
			wantNextLeaderTime: 1557906284061,
		},
		{
			desc:               "best block height equals to start block height",
			startTime:          1557906284061,
			now:                1557906284061,
			nodeOrder:          1,
			wantError:          nil,
			wantNextLeaderTime: 1557906284061 + BlockNumEachNode*BlockTimeInterval,
		},
		{
			desc:               "the node is producting block",
			startTime:          1557906284061,
			now:                1557906284561,
			nodeOrder:          0,
			wantError:          nil,
			wantNextLeaderTime: 1557906284061,
		},
		{
			desc:               "the node is producting block",
			startTime:          1557906284061,
			now:                1557906317561,
			nodeOrder:          1,
			wantError:          nil,
			wantNextLeaderTime: 1557906284061 + 66*BlockTimeInterval,
		},
		{
			desc:               "first round, must exclude genesis block",
			startTime:          1557906284061,
			now:                1557906286561,
			nodeOrder:          3,
			wantError:          nil,
			wantNextLeaderTime: 1557906284061 + 9*BlockTimeInterval,
		},
	}

	for i, c := range cases {
		nextLeaderTimestamp, err := nextLeaderTimeHelper(c.startTime, c.now, c.nodeOrder)
		if err != c.wantError {
			t.Fatalf("case #%d (%s) want error:%v, got error:%v", i, c.desc, c.wantError, err)
		}

		if err != nil {
			continue
		}
		if nextLeaderTimestamp != c.wantNextLeaderTime {
			t.Errorf("case #%d (%s) want next leader time:%d, got next leader time:%d", i, c.desc, c.wantNextLeaderTime, nextLeaderTimestamp)
		}
	}
}
