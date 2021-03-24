package upload

import (
	"strconv"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	log "github.com/sirupsen/logrus"

	"github.com/bytom/vapor/protocol/bc/types"
	"github.com/bytom/vapor/toolbar/apinode"
	"github.com/bytom/vapor/toolbar/osssync/util"
)

const LOCALDIR = "./blocks/"  // Local directory to store temp blocks files

// UploadKeeper the struct for upload
type UploadKeeper struct {
	Node      *apinode.Node
	OssClient *oss.Client
	OssBucket *oss.Bucket
	FileUtil  *util.FileUtil
}

// NewUploadKeeper return one new instance of UploadKeeper
func NewUploadKeeper() (*UploadKeeper, error) {
	cfg := &Config{}
	err := LoadConfig(&cfg)
	if err != nil {
		return nil, err
	}

	node := apinode.NewNode(cfg.VaporURL)

	ossClient, err := oss.New(cfg.OssConfig.Login.Endpoint, cfg.OssConfig.Login.AccessKeyID, cfg.OssConfig.Login.AccessKeySecret)
	if err != nil {
		return nil, err
	}

	ossBucket, err := ossClient.Bucket(cfg.OssConfig.Bucket)
	if err != nil {
		return nil, err
	}

	fileUtil := util.NewFileUtil(LOCALDIR)

	return &UploadKeeper{
		Node:      node,
		OssClient: ossClient,
		OssBucket: ossBucket,
		FileUtil:  fileUtil,
	}, nil
}

// RunSyncUp run synchronize upload to OSS
func (u *UploadKeeper) RunSyncUp() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for ; true; <-ticker.C {
		err := u.Upload()
		if err != nil {
			log.WithField("error", err).Errorln("blockKeeper fail on process block")
		}
	}
}

// Upload find and upload blocks
func (u *UploadKeeper) Upload() error {
	err := u.FileUtil.BlockDirInitial()
	if err != nil {
		return err
	}

	currBlockHeight, err := u.Node.GetBlockCount() // Current block height on vapor
	if err != nil {
		return err
	}

	infoJson, err := u.GetInfoJson()
	if err != nil {
		return err
	}

	latestUp := infoJson.LatestBlockHeight // Latest uploaded block height
	intervals := infoJson.Interval         // Interval array

	var pos1, pos2 int // currBlockHeight interval, latestUp interval
	for pos1 = len(intervals) - 1; currBlockHeight < intervals[pos1].StartBlockHeight; pos1-- {
	}
	// Current Block Height is out of the range given by info.json
	if currBlockHeight > intervals[pos1].EndBlockHeight {
		currBlockHeight = intervals[pos1].EndBlockHeight // Upload the part which contained by info.json
	}
	for pos2 = pos1; latestUp < intervals[pos2].StartBlockHeight; pos2-- {
	}

	// Upload Whole Interval
	for latestUp+1 < intervals[pos1].StartBlockHeight {
		err = u.UploadFiles(latestUp+1, intervals[pos2].EndBlockHeight, intervals[pos2].GzSize)
		if err != nil {
			return err
		}

		latestUp = intervals[pos2].EndBlockHeight
		pos2++
	}

	// Upload the last Interval
	newLatestUp := currBlockHeight - ((currBlockHeight - intervals[pos1].StartBlockHeight) % intervals[pos1].GzSize) - 1
	if latestUp < newLatestUp {
		err = u.UploadFiles(latestUp+1, newLatestUp, intervals[pos1].GzSize)
		if err != nil {
			return err
		}
	}
	return err
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

		_, err = u.FileUtil.SaveBlockFile(filename, blocks)
		if err != nil {
			return err
		}

		err = u.FileUtil.GzipCompress(filename)
		if err != nil {
			return err
		}

		err = u.OssBucket.PutObjectFromFile(filenameGzip, u.FileUtil.LocalDir+filenameGzip)
		if err != nil {
			return err
		}

		err = u.SetLatestBlockHeight(start + size - 1)
		if err != nil {
			return err
		}

		err = u.FileUtil.RemoveLocal(filenameJson)
		if err != nil {
			return err
		}

		err = u.FileUtil.RemoveLocal(filenameGzip)
		if err != nil {
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
	}
	return data, nil
}
