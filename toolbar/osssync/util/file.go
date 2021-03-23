package util

import "os"

// FileUtil is a struct of File utility
type FileUtil struct {
	LocalDir string
}

// IsExists if file or directory exist
func IsExists(path string) bool {
	_, err := os.Stat(path)
	if err != nil && !os.IsExist(err) {
		return false
	}
	return true
}

// PathExists return if path exists
func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}

	return false, err
}

// RemoveLocal deletes file
func (f *FileUtil) RemoveLocal(filename string) error {
	return os.Remove(f.LocalDir + filename)
}

// BlockDirInitial initializes the blocks directory
func (f *FileUtil) BlockDirInitial() error {
	ifPathExist, err := PathExists(f.LocalDir)
	if err != nil {
		return err
	}

	if ifPathExist {
		err = os.RemoveAll(f.LocalDir)
		if err != nil {
			return err
		}
	}

	err = os.Mkdir(f.LocalDir, 0755)
	return err
}
