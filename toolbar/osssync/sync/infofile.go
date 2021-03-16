package sync

import "github.com/bytom/vapor/toolbar/osssync/util"

// Interval is a struct for Interval list in info.json
type Interval struct {
	StartBlockHeight uint64
	EndBlockHeight   uint64
	GzSize           uint64
}

// NewInterval creates a new Interval from info.json
func NewInterval(start, intervalEnd, fileLength uint64) *Interval {
	return &Interval{
		StartBlockHeight: start,
		EndBlockHeight:   intervalEnd,
		GzSize:           fileLength,
	}
}

// Info is a struct for info.json
type Info struct {
	LatestBlockHeight uint64
	Interval          []*Interval
}

// NewInfo creates a new Info for info.json
func NewInfo(intervalEnd, fileLength uint64) *Info {
	newInvl := NewInterval(0, intervalEnd, fileLength)
	var arr []*Interval
	arr = append(arr, newInvl)
	return &Info{0, arr}
}

// GetInfoJson Download info.json
func (b *BlockKeeper) GetInfoJson() (*Info, error) {
	data, err := b.GetObjToData("info.json")
	if err != nil {
		return nil, err
	}

	info := new(Info)
	err = util.Json2Struct(data, &info)
	return info, err
}

// Upload info.json
func (b *BlockKeeper) PutInfoJson(infoData *Info) error {
	jsonData, err := util.Struct2Json(infoData)
	if err != nil {
		return err
	}

	// Upload
	return b.PutObjByteArr("info.json", jsonData)
}

// SetLatestBlockHeight set new latest blockHeight
func (b *BlockKeeper) SetLatestBlockHeight(newLatestBlockHeight uint64) error {
	info, err := b.GetInfoJson()
	if err != nil {
		return err
	}

	info.LatestBlockHeight = newLatestBlockHeight
	return b.PutInfoJson(info)
}

// AddInterval adds an interval to the end of info.json
// [{start: 0, end: 199999, size: 200000},{start: 200000, end: 19999999, size: 100000}]
func (b *BlockKeeper) AddInterval(intervalEnd, fileLength uint64) error {
	isJsonExisr, err := b.OssBucket.IsObjectExist("info.json")
	if err != nil {
		return err
	}

	var info *Info
	var newStart uint64

	if isJsonExisr {
		// Download info.json
		info, err = b.GetInfoJson()
		if err != nil {
			return err
		}

		// Add Interval
		lastInvl := info.Interval[len(info.Interval)-1]
		newStart = lastInvl.EndBlockHeight + 1
		newInvl := NewInterval(newStart, intervalEnd, fileLength)
		info.Interval = append(info.Interval, newInvl)
	} else {
		info = NewInfo(intervalEnd, fileLength)
		newStart = 0
	}

	return b.PutInfoJson(info)
}
