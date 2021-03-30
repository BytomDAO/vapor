package util

import (
	"compress/gzip"
	"io/ioutil"
	"os"
)

// GzipCompress Encode file to Gzip and save to the same directory
func (f *FileUtil) GzipCompress(fileName string) error {
	fw, err := os.Create(f.LocalDir + fileName + ".json.gz")
	if err != nil {
		return err
	}

	defer fw.Close()

	gw := gzip.NewWriter(fw)
	defer gw.Close()

	fr, err := os.Open(f.LocalDir + fileName + ".json")
	if err != nil {
		return err
	}

	defer fr.Close()

	fi, err := fr.Stat()
	if err != nil {
		return err
	}

	gw.Header.Name = fi.Name()

	buf := make([]byte, fi.Size())
	if _, err = fr.Read(buf); err != nil {
		return err
	}

	if _, err = gw.Write(buf); err != nil {
		return err
	}

	return nil
}

// GzipDecode Decode Gzip file and save to the same directory
func (f *FileUtil) GzipDecode(fileName string) error {
	fr, err := os.Open(f.LocalDir + fileName + ".json.gz")
	if err != nil {
		return err
	}

	defer fr.Close()

	reader, err := gzip.NewReader(fr)
	if err != nil {
		return err
	}

	defer reader.Close()

	json, err := ioutil.ReadAll(reader)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(f.LocalDir+fileName+".json", json, 0644)
}
