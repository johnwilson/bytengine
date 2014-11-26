package bytengine

type StateStore interface {
	TokenSet(token, user string, timeout int64) error
	TokenGet(token string) (string, error)
	CacheSet(id, value string, timeout int64) error
	CacheGet(id string) (string, error)
	ClearAll() error
	Start(config string) error
}
