package sync

import (
	"bytes"
	"io/ioutil"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

// PutObjByteArr upload Byte Array object
func (s *Sync) PutObjByteArr(objectName string, objectValue []byte) error {
	objectAcl := oss.ObjectACL(oss.ACLPublicRead)
	return s.OssBucket.PutObject(objectName, bytes.NewReader(objectValue), objectAcl)
}

// GetObjToData download object to stream
func (s *Sync) GetObjToData(objectName string) ([]byte, error) {
	body, err := s.OssBucket.GetObject(objectName)
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
