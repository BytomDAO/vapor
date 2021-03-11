package sync

import (
	"fmt"
	"github.com/bytom/vapor/toolbar/osssync/clients"
	"github.com/bytom/vapor/toolbar/osssync/config"
	"github.com/bytom/vapor/toolbar/osssync/util"
	"os"
	"strconv"
)

type InfoJson struct {
	Start uint64
	End uint64
	Size int
}

const LOCALDIR = "./blocks"

func mkDir() error {
	err := os.Mkdir(LOCALDIR,0755)
	return err
}

func UploadFiles(vaporClient *clients.VaporClient, ossBucket *clients.OssBucket, fileUtil *util.FileUtil, LOCALDIR string, start, end, size uint64) error {
	for  {
		if start > end {
			break
		}
		blocks, err := vaporClient.GetRawBlockArrayByBlockHeight(start, size)
		if err != nil {
			return err
		}
		filename := strconv.FormatUint(start, 10)
		_, err = fileUtil.SaveBlockFile(filename, blocks)
		if err != nil {
			return err
		}
		err = fileUtil.GzipCompress(filename)
		if err != nil {
			return err
		}
		err = ossBucket.PutObjLocal(filename + ".json.gz", LOCALDIR)
		if err != nil {
			return err
		}
		err = util.SetLatestBlockHeight(ossBucket, start + size - 1)
		if err != nil {
			return err
		}
		err = fileUtil.RemoveLocal(filename + ".json")
		if err != nil {
			return err
		}
		err = fileUtil.RemoveLocal(filename + ".json.gz")
		if err != nil {
			return err
		}
		start += size
	}
	return nil
}

func Upload(cfg *config.Config) error {
	//mkDir()
	ossClient, err := clients.NewOssClient(&cfg.Oss)
	if err != nil {
		return err
	}
	ossBucket, err := ossClient.AccessBucket("bytom-seed")
	if err != nil {
		return err
	}


	vaporClient := clients.NewVaporClient()
	currBlockHeight, err := vaporClient.GetBlockCount()  // Current block height on vapor
	if err != nil {
		return err
	}

	infoJson, err := util.GetInfoJson(ossBucket)
	if err != nil {
		return err
	}
	latestUp := infoJson.LatestBlockHeight  // Latest uploaded block height
	intervals := infoJson.Interval  // Interval array

	var pos1, pos2 int  // currBlockHeight interval, latestUp interval
	for pos1 = len(intervals) - 1; currBlockHeight < intervals[pos1].StartBlockHeight; pos1 -- {}
	// Current Block Height is out of the range given by info.json
	if currBlockHeight > intervals[pos1].EndBlockHeight {
		fmt.Println("Current Block Height is out of the range given by info.json")
		currBlockHeight = intervals[pos1].EndBlockHeight  // 只上传info.json包含的范围
	}
	for pos2 = pos1; latestUp < intervals[pos2].StartBlockHeight; pos2 -- {}

	// Upload
	fileUtil := util.NewFileUtil()
	for latestUp + 1 < intervals[pos1].StartBlockHeight {
		if latestUp == 0 {
			err = UploadFiles(vaporClient, ossBucket, fileUtil, LOCALDIR, latestUp, intervals[pos2].EndBlockHeight, intervals[pos2].GzSize)
		} else {
			err = UploadFiles(vaporClient, ossBucket, fileUtil, LOCALDIR, latestUp + 1, intervals[pos2].EndBlockHeight, intervals[pos2].GzSize)
		}
		if err != nil {
			return err
		}
		latestUp = intervals[pos2].EndBlockHeight
		pos2 ++
	}

	newLatestUp := currBlockHeight - ((currBlockHeight - intervals[pos1].StartBlockHeight) % intervals[pos1].GzSize) - 1
	if latestUp < newLatestUp {
		if latestUp == 0 {
			err = UploadFiles(vaporClient, ossBucket, fileUtil, LOCALDIR, latestUp, newLatestUp, intervals[pos1].GzSize)
		} else {
			err = UploadFiles(vaporClient, ossBucket, fileUtil, LOCALDIR, latestUp + 1, newLatestUp, intervals[pos1].GzSize)
		}
		if err != nil {
			return err
		}
	}
	return err
}


func CreateInfoJson(cfg *config.Config) error {
	ossClient, err := clients.NewOssClient(&cfg.Oss)
	if err != nil {
		return err
	}
	ossBucket, err := ossClient.AccessBucket("bytom-seed")
	if err != nil {
		return err
	}
	err = util.AddInterval(ossBucket, 59999999, 150000)
	if err != nil {
		return err
	}
	return err
}