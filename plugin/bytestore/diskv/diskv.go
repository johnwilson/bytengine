package diskv

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	bst "github.com/johnwilson/bytengine/bytestore"
	"github.com/johnwilson/bytengine/plugin"
	"github.com/nu7hatch/gouuid"
	"github.com/peterbourgon/diskv"
)

type Config struct {
	RootDir   string `json:"rootdir"`
	CacheSize uint64 `json:"cachesize"`
}

const SEP_CHAR = "-"

type DiskVBST struct {
	RootDir   string
	CacheSize uint64
	Transform func(s string) []string
	DB        *diskv.Diskv
}

func (m *DiskVBST) getKey(db, filename string) string {
	return db + SEP_CHAR + filename
}

func (m *DiskVBST) newKey(db string) (key string, id string) {
	tmp, err := uuid.NewV4()
	if err != nil {
		return "", ""
	}
	id = strings.Replace(tmp.String(), "-", "", -1)
	key = db + SEP_CHAR + id
	return
}

func (m *DiskVBST) save(key string, file *os.File) (map[string]interface{}, error) {
	defer file.Close()
	err := m.DB.WriteStream(key, file, false)
	if err != nil {
		return nil, err
	}

	info, err := bst.GetFileInfo(file.Name())
	if err != nil {
		return nil, err
	}

	return info, nil
}

func (m *DiskVBST) Start(config string) error {
	var c Config
	err := json.Unmarshal([]byte(config), &c)
	if err != nil {
		return err
	}

	transformFunc := func(s string) []string {
		return strings.Split(s, SEP_CHAR)
	}
	m.DB = diskv.New(diskv.Options{
		BasePath:     c.RootDir,
		Transform:    transformFunc,
		CacheSizeMax: 1024 * 1024 * c.CacheSize, // in megabytes
	})
	return nil
}

func (m *DiskVBST) Add(db string, file *os.File) (map[string]interface{}, error) {
	defer file.Close()
	key, filename := m.newKey(db)
	if len(key) == 0 {
		return nil, fmt.Errorf("Item could not be added: invalid key")
	}

	info, err := m.save(key, file)
	if err != nil {
		return nil, err
	}
	info["name"] = filename
	return info, nil
}

func (m *DiskVBST) Update(db, filename string, file *os.File) (map[string]interface{}, error) {
	defer file.Close()
	key := m.getKey(db, filename)
	info, err := m.save(key, file)
	if err != nil {
		return nil, err
	}
	return info, nil
}

func (m *DiskVBST) Delete(db, filename string) error {
	key := m.getKey(db, filename)
	err := m.DB.Erase(key)
	if err != nil {
		return err
	}
	return nil
}

func (m *DiskVBST) Read(db, filename string, file io.Writer) error {
	out, err := m.DB.ReadStream(m.getKey(db, filename), true)
	if err != nil {
		return err
	}
	defer out.Close()

	b := make([]byte, 1024)
	for {
		n, err := out.Read(b)
		if err != nil && err != io.EOF {
			return err
		}
		if n == 0 {
			break
		}

		_, err = file.Write(b[:n])
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *DiskVBST) DropDatabase(db string) error {
	for key := range m.DB.Keys() {
		prefix := db + SEP_CHAR
		if strings.HasPrefix(key, prefix) {
			err := m.DB.Erase(key)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func NewDiskVBST() *DiskVBST {
	return &DiskVBST{}
}

func init() {
	plugin.Register("diskv", NewDiskVBST())
}
