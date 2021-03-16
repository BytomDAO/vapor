package util

import (
	"compress/gzip"
	"os"
)

const READ_SIZE = 1024 * 1024 * 500

// GzipCompress compress file to Gzip
func (f *FileUtil) GzipCompress(fileName string) error {
	filePath := f.LocalDir + "/" + fileName + ".json.gz"
	fw, err := os.Create(filePath)
	if err != nil {
		return err
	}

	defer fw.Close()

	gw := gzip.NewWriter(fw)
	defer gw.Close()

	filePath = f.LocalDir + "/" + fileName + ".json"
	fr, err := os.Open(filePath)
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
	_, err = fr.Read(buf)
	if err != nil {
		return err
	}

	_, err = gw.Write(buf)
	if err != nil {
		return err
	}

	return err
}

// GzipUncompress uncompress Gzip file
func (f *FileUtil) GzipUncompress(fileName string) error {
	filedirname := f.LocalDir + "/" + fileName + ".json.gz"
	fr, err := os.Open(filedirname)
	if err != nil {
		return err
	}

	defer fr.Close()

	gr, err := gzip.NewReader(fr)
	if err != nil {
		return err
	}

	defer gr.Close()

	buf := make([]byte, READ_SIZE)
	n, err := gr.Read(buf)

	filedirname = f.LocalDir + "/" + gr.Header.Name
	fw, err := os.Create(filedirname)
	if err != nil {
		return err
	}

	_, err = fw.Write(buf[:n])
	if err != nil {
		return err
	}

	return err
}
