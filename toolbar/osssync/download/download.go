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
	err := d.FileUtil.BlockDirInitial()
	if err != nil {
		return err
	}

	latestDown := d.Node.GetChain().BestBlockHeight() // latest block height on local node

	infoJson, err := d.GetInfoJson()
	if err != nil {
		return err
	}

	latestUp := infoJson.LatestBlockHeight // Latest uploaded block height on OSS
	intervals := infoJson.Interval         // Interval array

	var pos1, pos2 int // latestDown interval, latestUp interval
	for pos1 = len(intervals) - 1; latestDown < intervals[pos1].StartBlockHeight; pos1-- {
	}
	for pos2 = pos1; latestUp > intervals[pos2].EndBlockHeight; pos2++ {
	}

	for pos1 < pos2 {
		err = d.DownloadFiles(latestDown+1, intervals[pos1].EndBlockHeight, intervals[pos1].GzSize)
		if err != nil {
			return err
		}
		pos1++
	}
	if pos1 == pos2 {
		err = d.DownloadFiles(intervals[pos2].StartBlockHeight, latestUp, intervals[pos2].GzSize)
		if err != nil {
			return err
		}
	}
	return err
}

// DownloadFiles get block files from OSS, and update the node
func (d *DownloadKeeper) DownloadFiles(start, end, size uint64) error {
	for {
		if start > end {
			break
		}

		filename := strconv.FormatUint(start, 10)
		filenameJson := filename + ".json"
		filenameGzip := filenameJson + ".gz"

		err := d.GetObjectToFile(filenameGzip)
		if err != nil {
			return err
		}

		err = d.FileUtil.GzipUncompress(filename)
		if err != nil {
			return err
		}

		blocksJson, err := d.FileUtil.GetJson(filenameJson)
		if err != nil {
			return err
		}

		blocks := []*types.Block{}
		err = util.Json2Struct(blocksJson, blocks)
		if err != nil {
			return err
		}

		latestDown := d.Node.GetChain().BestBlockHeight()

		if latestDown+1 > start {
			blocks = blocks[latestDown-start:] // start from latestDown+1
		}

		err = d.SyncToNode(blocks)
		if err != nil {
			return err
		}

		err = d.FileUtil.RemoveLocal(filenameGzip)
		if err != nil {
			return err
		}

		err = d.FileUtil.RemoveLocal(filenameJson)
		if err != nil {
			return err
		}

		start += size
	}
	return nil
}

// SyncToNode synchronize blocks to local node
func (d *DownloadKeeper) SyncToNode(blocks []*types.Block) error {
	for i := 0; i < len(blocks); i++ {
		_, err := d.Node.GetChain().ProcessBlock(blocks[i])
		if err != nil {
			return err
		}
	}
	return nil
}
