package bytengine

type DataFilter interface {
	Start(config string) error
	Apply(filter string, r *Response, eng *Engine) Response
	Info(filter string) string
	Check(filter string) bool
	All() []string
}
