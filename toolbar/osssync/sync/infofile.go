package sync

import "github.com/bytom/vapor/toolbar/osssync/util"

// Interval determines the number of blocks in a Gzip file in the Interval of blockHeight
// StartBlockHeight is the start of the Interval
// EndBlockHeight: the end of the Interval
// GzSize is the number of blocks store in a Gzip file
type Interval struct {
	StartBlockHeight uint64
	EndBlockHeight   uint64
	GzSize           uint64
}

// NewInterval creates a new Interval from info.json
func NewInterval(start, end, gzSize uint64) *Interval {
	return &Interval{
		StartBlockHeight: start,
		EndBlockHeight:   end,
		GzSize:           gzSize,
	}
}

// Info is a struct for info.json
type Info struct {
	LatestBlockHeight uint64
	Interval          []*Interval
}

// NewInfo creates a new Info for info.json
func NewInfo(end, gzSize uint64) *Info {
	newInvl := NewInterval(0, end, gzSize)
	var arr []*Interval
	arr = append(arr, newInvl)
	return &Info{0, arr}
}

// GetInfoJson Download info.json
func (b *Sync) GetInfoJson() (*Info, error) {
	data, err := b.GetObjToData("info.json")
	if err != nil {
		return nil, err
	}

	info := new(Info)
	err = util.Json2Struct(data, &info)
	return info, err
}

// Upload info.json
func (b *Sync) PutInfoJson(infoData *Info) error {
	jsonData, err := util.Struct2Json(infoData)
	if err != nil {
		return err
	}

	// Upload
	return b.PutObjByteArr("info.json", jsonData)
}

// SetLatestBlockHeight set new latest blockHeight on OSS
func (b *Sync) SetLatestBlockHeight(newLatestBlockHeight uint64) error {
	info, err := b.GetInfoJson()
	if err != nil {
		return err
	}

	info.LatestBlockHeight = newLatestBlockHeight
	return b.PutInfoJson(info)
}

// AddInterval adds an interval to the end of info.json
func (b *Sync) AddInterval(end, gzSize uint64) error {
	isJsonExist, err := b.OssBucket.IsObjectExist("info.json")
	if err != nil {
		return err
	}

	var info *Info
	if isJsonExist {
		// Download info.json
		info, err = b.GetInfoJson()
		if err != nil {
			return err
		}

		// Add Interval
		prevInvl := info.Interval[len(info.Interval)-1]
		newInvl := NewInterval(prevInvl.EndBlockHeight+1, end, gzSize)
		info.Interval = append(info.Interval, newInvl)
	} else {
		info = NewInfo(end, gzSize)
	}
	return b.PutInfoJson(info)
}
