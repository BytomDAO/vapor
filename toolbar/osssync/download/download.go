package download

import (
	"strconv"

	"github.com/bytom/vapor/errors"
	"github.com/bytom/vapor/node"
	"github.com/bytom/vapor/protocol/bc/types"
	"github.com/bytom/vapor/toolbar/osssync/util"
)

const LOCALDIR = "./toolbar/osssync/blocks/" // Local directory to store temp blocks files

// Run synchronize download from OSS to local node
func Run(node *node.Node, ossEndpoint string) error {
	if ossEndpoint == "" {
		return errors.New("OSS Endpoint is empty")
	}

	downloadKeeper, err := NewDownloadKeeper(node, ossEndpoint)
	if err != nil {
		return err
	}

	if err = downloadKeeper.Download(); err != nil {
		return err
	}

	return nil
}

// DownloadKeeper the struct for download
type DownloadKeeper struct {
	Node        *node.Node
	OssEndpoint string
	FileUtil    *util.FileUtil
}

// NewDownloadKeeper return one new instance of DownloadKeeper
func NewDownloadKeeper(node *node.Node, ossEndpoint string) (*DownloadKeeper, error) {
	fileUtil := util.NewFileUtil(LOCALDIR)

	return &DownloadKeeper{
		Node:        node,
		OssEndpoint: "http://" + ossEndpoint + "/",
		FileUtil:    fileUtil,
	}, nil
}

// Download get blocks from OSS and update the node
func (d *DownloadKeeper) Download() error {
	if err := d.FileUtil.BlockDirInitial(); err != nil {
		return err
	}

	syncStart := d.Node.GetChain().BestBlockHeight() + 1 // block height which the synchronization start from

	infoJson, err := d.GetInfoJson()
	if err != nil {
		return err
	}

	latestUp := infoJson.LatestBlockHeight // Latest uploaded block height on OSS
	intervals := infoJson.Interval         // Interval array
	if latestUp == 0 || latestUp < syncStart {
		return errors.New("No new blocks on OSS.")
	}

	var pos1, pos2 int // syncStart interval, latestUp interval
	// Find pos2
	for pos2 = len(intervals) - 1; latestUp < intervals[pos2].StartBlockHeight; pos2-- {
	}
	// Find pos1
	if syncStart == 0 {
		pos1 = 0
	} else {
		for pos1 = pos2; syncStart < intervals[pos1].StartBlockHeight; pos1-- {
		}
	}

	// Download Whole Interval
	for pos1 < pos2 {
		if err = d.DownloadFiles(syncStart, intervals[pos1].EndBlockHeight, intervals[pos1]); err != nil {
			return err
		}
		syncStart = intervals[pos1].EndBlockHeight + 1
		pos1++
	}
	// Download the last Interval
	if pos1 == pos2 {
		if err = d.DownloadFiles(syncStart, latestUp, intervals[pos2]); err != nil {
			return err
		}
	}
	return nil
}

// DownloadFiles get block files from OSS, and update the node
func (d *DownloadKeeper) DownloadFiles(start, end uint64, interval *util.Interval) error {
	size := interval.GzSize
	for {
		if start > end {
			break
		}

		intervalStart := interval.StartBlockHeight
		startInFile := start - intervalStart
		n := startInFile / size
		filenameNum := n*size + intervalStart

		filename := strconv.FormatUint(filenameNum, 10)
		filenameJson := filename + ".json"
		filenameGzip := filenameJson + ".gz"

		if err := d.GetObjectToFile(filenameGzip); err != nil {
			return err
		}

		if err := d.FileUtil.GzipDecode(filename); err != nil {
			return err
		}

		if err := d.FileUtil.RemoveLocal(filenameGzip); err != nil {
			return err
		}

		blocksJson, err := d.FileUtil.GetJson(filenameJson)
		if err != nil {
			return err
		}

		blocks := []*types.Block{}
		if err = util.Json2Struct(blocksJson, &blocks); err != nil {
			return err
		}

		latestDown := d.Node.GetChain().BestBlockHeight()
		if latestDown+1 > start {
			blocks = blocks[startInFile:] // start from latestDown+1
		} else if latestDown+1 < start {
			return errors.New("Wrong interval")
		}
		if err = d.SyncToNode(blocks); err != nil {
			return err
		}

		if err = d.FileUtil.RemoveLocal(filenameJson); err != nil {
			return err
		}

		start = filenameNum + size
	}
	return nil
}

// SyncToNode synchronize blocks to local node
func (d *DownloadKeeper) SyncToNode(blocks []*types.Block) error {
	for i := 0; i < len(blocks); i++ {
		if _, err := d.Node.GetChain().ProcessBlock(blocks[i]); err != nil {
			return err
		}
	}
	return nil
}
