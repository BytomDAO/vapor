package upload

import (
	"strconv"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	log "github.com/sirupsen/logrus"

	"github.com/bytom/vapor/errors"
	"github.com/bytom/vapor/protocol/bc/types"
	"github.com/bytom/vapor/toolbar/apinode"
	"github.com/bytom/vapor/toolbar/osssync/util"
)

const LOCALDIR = "./blocks/" // Local directory to store temp blocks files

// Run synchronize upload blocks from vapor to OSS
func Run() error {
	uploadKeeper, err := NewUploadKeeper()
	if err != nil {
		return err
	}

	uploadKeeper.Run()
	return nil
}

// AddInterval if "info.json" exists on OSS, add Interval to the end; if not exist, create "info.json" with Interval
func AddInterval(end, gzSize uint64) error {
	uploadKeeper, err := NewUploadKeeper()
	if err != nil {
		return err
	}

	return uploadKeeper.AddInterval(end, gzSize)
}

// UploadKeeper the struct for upload
type UploadKeeper struct {
	Node      *apinode.Node
	OssClient *oss.Client
	OssBucket *oss.Bucket
	OssDir    string
	FileUtil  *util.FileUtil
}

// NewUploadKeeper return one new instance of UploadKeeper
func NewUploadKeeper() (*UploadKeeper, error) {
	cfg := &Config{}
	if err := LoadConfig(&cfg); err != nil {
		return nil, err
	}

	ossClient, err := oss.New(cfg.OssConfig.Login.Endpoint, cfg.OssConfig.Login.AccessKeyID, cfg.OssConfig.Login.AccessKeySecret)
	if err != nil {
		return nil, err
	}

	ossBucket, err := ossClient.Bucket(cfg.OssConfig.Bucket)
	if err != nil {
		return nil, err
	}

	return &UploadKeeper{
		Node:      apinode.NewNode(cfg.VaporURL),
		OssClient: ossClient,
		OssBucket: ossBucket,
		OssDir:    cfg.OssConfig.Directory,
		FileUtil:  util.NewFileUtil(LOCALDIR),
	}, nil
}

// Run synchronize upload blocks from vapor to OSS
func (u *UploadKeeper) Run() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for ; true; <-ticker.C {
		if err := u.Upload(); err != nil {
			log.WithField("error", err).Errorln("blockKeeper fail")
		}
	}
}

// Upload find and upload blocks
func (u *UploadKeeper) Upload() error {
	if err := u.FileUtil.BlockDirInitial(); err != nil {
		return err
	}

	currBlockHeight, err := u.Node.GetBlockCount() // Current block height on vapor
	if err != nil {
		return err
	}

	if currBlockHeight == 0 {
		return errors.New("Current block height is 0.")
	}

	infoJson, err := u.GetInfoJson()
	if err != nil {
		return err
	}

	latestUp := infoJson.LatestBlockHeight // Latest uploaded block height
	intervals := infoJson.Interval         // Interval array

	var pos1, pos2 int // currBlockHeight interval, latestUp interval
	// Find pos1
	for pos1 = len(intervals) - 1; currBlockHeight < intervals[pos1].StartBlockHeight; pos1-- {
	}
	// Current Block Height is out of the range given by info.json
	if currBlockHeight > intervals[pos1].EndBlockHeight {
		currBlockHeight = intervals[pos1].EndBlockHeight // Upload the part which contained by info.json
	}
	// Find pos2
	if latestUp == 0 {
		pos2 = 0
	} else {
		for pos2 = pos1; latestUp < intervals[pos2].StartBlockHeight; pos2-- {
		}
	}

	// Upload Whole Interval
	for latestUp+1 < intervals[pos1].StartBlockHeight {
		if err = u.UploadFiles(latestUp+1, intervals[pos2].EndBlockHeight, intervals[pos2].GzSize); err != nil {
			return err
		}

		latestUp = intervals[pos2].EndBlockHeight
		pos2++
	}

	// Upload the last Interval
	newLatestUp := currBlockHeight - ((currBlockHeight - intervals[pos1].StartBlockHeight + 1) % intervals[pos1].GzSize)
	if latestUp < newLatestUp {
		if err = u.UploadFiles(latestUp+1, newLatestUp, intervals[pos1].GzSize); err != nil {
			return err
		}
	}
	return nil
}

// UploadFiles get block from vapor and upload files to OSS
func (u *UploadKeeper) UploadFiles(start, end, size uint64) error {
	for {
		if start > end {
			break
		}
		blocks, err := u.GetBlockArray(start, size)
		if err != nil {
			return err
		}

		filename := strconv.FormatUint(start, 10)
		filenameJson := filename + ".json"
		filenameGzip := filenameJson + ".gz"

		if _, err = u.FileUtil.SaveBlockFile(filename, blocks); err != nil {
			return err
		}

		if err = u.FileUtil.GzipCompress(filename); err != nil {
			return err
		}

		if err = u.FileUtil.RemoveLocal(filenameJson); err != nil {
			return err
		}

		if err = u.OssBucket.PutObjectFromFile(u.OssDir+filenameGzip, u.FileUtil.LocalDir+filenameGzip); err != nil {
			return err
		}

		if err = u.SetLatestBlockHeight(start + size - 1); err != nil {
			return err
		}

		if err = u.FileUtil.RemoveLocal(filenameGzip); err != nil {
			return err
		}

		start += size
	}
	return nil
}

// GetBlockArray return the RawBlockArray by BlockHeight from start to start+length-1
func (u *UploadKeeper) GetBlockArray(start, length uint64) ([]*types.Block, error) {
	blockHeight := start
	data := []*types.Block{}
	for i := uint64(0); i < length; i++ {
		resp, err := u.Node.GetBlockByHeight(blockHeight)
		if err != nil {
			return nil, err
		}

		data = append(data, resp)
		blockHeight++
		//time.Sleep(time.Duration(5) * time.Millisecond)
	}
	return data, nil
}
