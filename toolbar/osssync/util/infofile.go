package util

import (
	"github.com/bytom/vapor/toolbar/osssync/clients"
)

type Interval struct {
	StartBlockHeight	uint64
	EndBlockHeight		uint64
	GzSize				uint64
}

func NewInterval(start, intervalEnd, fileLength uint64) *Interval {
	return &Interval{
		StartBlockHeight: start,
		EndBlockHeight: intervalEnd,
		GzSize: fileLength,
	}
}

type Info struct {
	LatestBlockHeight	uint64
	Interval		[]*Interval
}

func NewInfo(intervalEnd, fileLength uint64) *Info {
	newInvl := NewInterval(0, intervalEnd, fileLength)
	var arr []*Interval
	arr = append(arr, newInvl)
	return &Info{0, arr}
}

// Download info.json
func GetInfoJson(b *clients.OssBucket) (*Info, error) {
	data, err := b.GetObjToData("info.json")
	if err != nil {
		return nil, err
	}
	info := new(Info)
	err = Json2Struct(data, &info)
	return info, err
}

// Upload info.json
func PutInfoJson(b *clients.OssBucket, infoData *Info) error {
	jsonData, err := Struct2Json(infoData)
	if err != nil {
		return err
	}
	// Upload
	return b.PutObjByteArr("info.json", jsonData)
}

func SetLatestBlockHeight(b *clients.OssBucket, newLatestBlockHeight uint64) error {
	info, err := GetInfoJson(b)
	if err != nil {
		return err
	}
	info.LatestBlockHeight = newLatestBlockHeight
	return PutInfoJson(b, info)
}

// AddInterval adds an interval to the end of info.json
// [{start: 0, end: 199999, size: 200000},{start: 200000, end: 19999999, size: 100000}]
func AddInterval(b *clients.OssBucket, intervalEnd, fileLength uint64) error {
	isJsonExisr, err := b.IsExist("info.json")
	if err != nil {
		return err
	}

	var info *Info
	var newStart uint64

	if isJsonExisr {
		// Download info.json
		info, err = GetInfoJson(b)
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

	return PutInfoJson(b, info)
}