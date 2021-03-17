package sync

import (
	"github.com/aliyun/aliyun-oss-go-sdk/oss"

	"github.com/bytom/vapor/toolbar/apinode"
	"github.com/bytom/vapor/toolbar/osssync/config"
	"github.com/bytom/vapor/toolbar/osssync/util"
)

// BlockKeeper the struct of the BlockKeeper
type BlockKeeper struct {
	Node      *apinode.Node
	OssClient *oss.Client
	OssBucket *oss.Bucket
	FileUtil  *util.FileUtil
}

// NewBlockKeeper return one new instance of BlockKeeper
func NewBlockKeeper() (*BlockKeeper, error) {
	cfg := &config.Config{}
	err := config.LoadConfig(&cfg)
	if err != nil {
		return nil, err
	}

	node := apinode.NewNode(cfg.VaporURL)

	ossClient, err := oss.New(cfg.Oss.Endpoint, cfg.Oss.AccessKeyID, cfg.Oss.AccessKeySecret)
	if err != nil {
		return nil, err
	}

	ossBucket, err := ossClient.Bucket("bytom-seed")
	if err != nil {
		return nil, err
	}

	fileUtil := util.NewFileUtil("./blocks")

	return &BlockKeeper{
		Node:      node,
		OssClient: ossClient,
		OssBucket: ossBucket,
		FileUtil:  fileUtil,
	}, nil
}
