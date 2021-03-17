package sync

import (
	log "github.com/sirupsen/logrus"
	"strconv"
	"time"
)

const HOUR = 3600 * 1000

// RunSyncUp run synchronize upload to OSS
func (b *BlockKeeper) RunSyncUp() {
	ticker := time.NewTicker(time.Duration(HOUR) * time.Millisecond)
	defer ticker.Stop()

	for ; true; <-ticker.C {
		err := b.Upload()
		if err != nil {
			log.WithField("error", err).Errorln("blockKeeper fail on process block")
		}
	}
}

// UploadFiles get block from vapor and upload files to OSS
func (b *BlockKeeper) UploadFiles(start, end, size uint64) error {
	for {
		if start > end {
			break
		}
		blocks, err := b.GetBlockArray(start, size)
		if err != nil {
			return err
		}

		filename := strconv.FormatUint(start, 10)
		filenameJson := filename + ".json"
		filenameGzip := filenameJson + ".gz"

		_, err = b.FileUtil.SaveBlockFile(filename, blocks)
		if err != nil {
			return err
		}

		err = b.FileUtil.GzipCompress(filename)
		if err != nil {
			return err
		}

		err = b.OssBucket.PutObjectFromFile(filenameGzip, b.FileUtil.LocalDir+"/"+filenameGzip)
		if err != nil {
			return err
		}

		err = b.SetLatestBlockHeight(start + size - 1)
		if err != nil {
			return err
		}

		err = b.FileUtil.RemoveLocal(filenameJson)
		if err != nil {
			return err
		}

		err = b.FileUtil.RemoveLocal(filename + ".json.gz")
		if err != nil {
			return err
		}

		start += size
	}
	return nil
}

// Upload upload blocks
func (b *BlockKeeper) Upload() error {
	b.FileUtil.BlockDirInitial()

	currBlockHeight, err := b.Node.GetBlockCount() // Current block height on vapor
	if err != nil {
		return err
	}

	infoJson, err := b.GetInfoJson()
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

	// Upload
	for latestUp+1 < intervals[pos1].StartBlockHeight {
		if latestUp == 0 {
			err = b.UploadFiles(latestUp, intervals[pos2].EndBlockHeight, intervals[pos2].GzSize)
		} else {
			err = b.UploadFiles(latestUp+1, intervals[pos2].EndBlockHeight, intervals[pos2].GzSize)
		}
		if err != nil {
			return err
		}
		latestUp = intervals[pos2].EndBlockHeight
		pos2++
	}

	newLatestUp := currBlockHeight - ((currBlockHeight - intervals[pos1].StartBlockHeight) % intervals[pos1].GzSize) - 1
	if latestUp < newLatestUp {
		if latestUp == 0 {
			err = b.UploadFiles(latestUp, newLatestUp, intervals[pos1].GzSize)
		} else {
			err = b.UploadFiles(latestUp+1, newLatestUp, intervals[pos1].GzSize)
		}
		if err != nil {
			return err
		}
	}
	return err
}
