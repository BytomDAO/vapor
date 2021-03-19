package sync

import (
	"github.com/aliyun/aliyun-oss-go-sdk/oss"

	"github.com/bytom/vapor/toolbar/osssync/config"
	"github.com/bytom/vapor/toolbar/osssync/util"
)

// Sync the struct of the Sync
type Sync struct {
	OssClient *oss.Client
	OssBucket *oss.Bucket
	FileUtil  *util.FileUtil
}

// NewSync return one new instance of Sync
func NewSync() (*Sync, error) {
	cfg := &config.Config{}
	err := config.LoadConfig(&cfg)
	if err != nil {
		return nil, err
	}

	ossClient, err := oss.New(cfg.Oss.Endpoint, cfg.Oss.AccessKeyID, cfg.Oss.AccessKeySecret)
	if err != nil {
		return nil, err
	}

	ossBucket, err := ossClient.Bucket("bytom-seed")
	if err != nil {
		return nil, err
	}

	fileUtil := util.NewFileUtil("./blocks")

	return &Sync{
		OssClient: ossClient,
		OssBucket: ossBucket,
		FileUtil:  fileUtil,
	}, nil
}
