package bytestore

import (
	"net/http"
	"os"
	"path"
	"strings"
)

// to be expanded
var MimeList = map[string]string{
	".js":  "text/javascript",
	".css": "text/css",
}

func GetFileInfo(fpath string) (map[string]interface{}, error) {
	// try and get uploaded file mime type
	tmpfile, err := os.Open(fpath)
	if err != nil {
		return nil, err
	}
	defer tmpfile.Close()

	// read 1024 bytes to enable mime type retrieval
	mimebuffer := make([]byte, 1024)
	_, err = tmpfile.Read(mimebuffer)
	if err != nil {
		return nil, err
	}
	mime := http.DetectContentType(mimebuffer)

	// if mime is 'text/plain' try and get exact mime from file extension
	prefix := "text/plain;"
	if strings.HasPrefix(mime, prefix) {
		ext := path.Ext(fpath)
		mval, exists := MimeList[ext]
		if exists {
			mime = strings.Replace(mime, prefix, mval, 1)
		}
	}

	// get total file size
	f_info, _ := tmpfile.Stat()
	size := f_info.Size()

	val := make(map[string]interface{})
	val["mime"] = mime
	val["size"] = size
	return val, nil
}
