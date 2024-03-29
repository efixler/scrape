package resource

type FetchMethod int

const (
	Client FetchMethod = iota
	Headless
)

func (f FetchMethod) String() string {
	switch f {
	case Client:
		return "client"
	case Headless:
		return "headless"
	default:
		return "unknown"
	}
}
