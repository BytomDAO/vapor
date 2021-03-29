package upload

import (
	"bytes"
	"github.com/bytom/vapor/errors"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"

	"github.com/bytom/vapor/toolbar/osssync/util"
)

// PutObjByteArr upload Byte Array object
func (u *UploadKeeper) PutObjByteArr(objectName string, objectValue []byte) error {
	objectAcl := oss.ObjectACL(oss.ACLPublicRead)
	return u.OssBucket.PutObject(objectName, bytes.NewReader(objectValue), objectAcl)
}

// GetInfoJson Download info.json
func (u *UploadKeeper) GetInfoJson() (*util.Info, error) {
	body, err := u.OssBucket.GetObject("info.json")
	if err != nil {
		return nil, err
	}

	return util.GetInfoJson(body)
}

// Upload info.json
func (u *UploadKeeper) PutInfoJson(infoData *util.Info) error {
	jsonData, err := util.Struct2Json(infoData)
	if err != nil {
		return err
	}

	// Upload
	return u.PutObjByteArr("info.json", jsonData)
}

// SetLatestBlockHeight set new latest blockHeight on OSS
func (u *UploadKeeper) SetLatestBlockHeight(newLatestBlockHeight uint64) error {
	info, err := u.GetInfoJson()
	if err != nil {
		return err
	}

	info.LatestBlockHeight = newLatestBlockHeight
	return u.PutInfoJson(info)
}

// AddInterval if "info.json" exists on OSS, add Interval to the end; if not exist, create "info.json" with Interval
func (u *UploadKeeper) AddInterval(end, gzSize uint64) error {
	isJsonExist, err := u.OssBucket.IsObjectExist("info.json")
	if err != nil {
		return err
	}

	var info *util.Info
	if isJsonExist {
		// Download info.json
		info, err = u.GetInfoJson()
		if err != nil {
			return err
		}

		// Add Interval
		prevInvl := info.Interval[len(info.Interval)-1]
		if prevInvl.EndBlockHeight >= end {
			return errors.New("New interval is included in previous intervals.")
		}

		if (end-prevInvl.EndBlockHeight)%gzSize != 0 {
			return errors.New("New interval is invalid.")
		}
		
		newInvl := util.NewInterval(prevInvl.EndBlockHeight+1, end, gzSize)
		info.Interval = append(info.Interval, newInvl)
	} else {
		info = util.NewInfo(end, gzSize)
	}
	return u.PutInfoJson(info)
}
