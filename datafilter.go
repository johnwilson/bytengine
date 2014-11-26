package bytengine

type DataFilter interface {
	Start(config string) error
	Apply(filter string, r *Response) Response
	Info(filter string) string
	Check(filter string) bool
	All() []string
}
