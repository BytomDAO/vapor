package download

import (
	"io"
	"net/http"
	"os"

	"github.com/bytom/vapor/toolbar/osssync/util"
)

// GetObject download the file object from OSS
func (d *DownloadKeeper) GetObject(filename string) (*io.ReadCloser, error) {
	url := d.OssEndpoint + filename
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	return &res.Body, nil
}

// GetObjectToFile download the file object from OSS to local
func (d *DownloadKeeper) GetObjectToFile(filename string) error {
	f, err := os.Create(d.FileUtil.LocalDir + filename)
	if err != nil {
		return err
	}

	body, err := d.GetObject(filename)
	if err != nil {
		return err
	}

	defer (*body).Close()

	io.Copy(f, *body)
	return nil
}

// GetInfoJson Download info.json
func (d *DownloadKeeper) GetInfoJson() (*util.Info, error) {
	body, err := d.GetObject("info.json")
	if err != nil {
		return nil, err
	}

	return util.GetInfoJson(*body)
}
