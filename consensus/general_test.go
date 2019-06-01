package consensus

import "testing"

func TestSubsidy(t *testing.T) {
	ActiveNetParams = Params{
		ProducerSubsidys: []ProducerSubsidy{
			{BeginBlock: 0, EndBlock: 0, Subsidy: 24},
			{BeginBlock: 1, EndBlock: 840000, Subsidy: 24},
			{BeginBlock: 840001, EndBlock: 1680000, Subsidy: 12},
			{BeginBlock: 1680001, EndBlock: 3360000, Subsidy: 6},
		},
	}
	subsidyReductionInterval := uint64(840000)
	cases := []struct {
		subsidy uint64
		height  uint64
	}{
		{
			subsidy: 24,
			height:  1,
		},
		{
			subsidy: 24,
			height:  subsidyReductionInterval - 1,
		},
		{
			subsidy: 24,
			height:  subsidyReductionInterval,
		},
		{
			subsidy: 12,
			height:  subsidyReductionInterval + 1,
		},
		{
			subsidy: 0,
			height:  subsidyReductionInterval * 10,
		},
	}

	for _, c := range cases {
		subsidy := BlockSubsidy(c.height)
		if subsidy != c.subsidy {
			t.Errorf("got subsidy %d, want %d", subsidy, c.subsidy)
		}
	}
}
