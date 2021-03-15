package clients

import (
	"bytes"
	"io/ioutil"
	"strings"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"

	"github.com/bytom/vapor/toolbar/osssync/config"
)

// OssClient OSS Client
type OssClient struct {
	*oss.Client
}

// NewOssClient creates a new OssClient
func NewOssClient(config *config.Oss) (*OssClient, error) {
	client, err := oss.New(config.Endpoint, config.AccessKeyID, config.AccessKeySecret)
	if err != nil {
		return nil, err
	}
	return &OssClient{client}, err
}

// OssBucket OSS Bucket
type OssBucket struct {
	*oss.Bucket
}

// AccessBucket creates a new access to bucket
func (c *OssClient) AccessBucket(bucketName string) (*OssBucket, error) {
	bucket, err := c.Bucket(bucketName)
	if err != nil {
		return nil, err
	}
	return &OssBucket{bucket}, err
}

// PutObjString upload String object
func (b *OssBucket) PutObjString(objectName, objectValue string) error {
	storageType := oss.ObjectStorageClass(oss.StorageStandard)
	objectAcl := oss.ObjectACL(oss.ACLPublicRead)
	return b.PutObject(objectName, strings.NewReader(objectValue), storageType, objectAcl)
}

// PutObjByteArr upload Byte Array object
func (b *OssBucket) PutObjByteArr(objectName string, objectValue []byte) error {
	objectAcl := oss.ObjectACL(oss.ACLPublicRead)
	return b.PutObject(objectName, bytes.NewReader(objectValue), objectAcl)
}

// PutObjFile upload Local File object
func (b *OssBucket) PutObjFile(objectName, localDir string) error {
	return b.PutObjectFromFile(objectName, localDir+"/"+objectName)
}

// DelObj deletes Object
func (b *OssBucket) DelObj(objectName string) error {
	return b.DeleteObject(objectName)
}

// GetObjToData download object to stream
func (b *OssBucket) GetObjToData(objectName string) ([]byte, error) {
	body, err := b.GetObject(objectName)
	if err != nil {
		return nil, err
	}
	defer body.Close()

	data, err := ioutil.ReadAll(body)
	if err != nil {
		return nil, err
	}
	return data, err
}

// GetObjToFile download object to local
func (b *OssBucket) GetObjToFile(objectName, localDir string) error {
	return b.GetObjectToFile(objectName, localDir+"/"+objectName)
}

// IsExist checks if the file exist
func (b *OssBucket) IsExist(objectName string) (bool, error) {
	return b.IsObjectExist(objectName)
}
