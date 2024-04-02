package ua

type UserAgent string

const (
	Firefox88 = UserAgent("Mozilla/5.0 (X11; Linux x86_64; rv:88.0) Gecko/20100101 Firefox/88.0")
	Safari537 = UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/77.0.3830.0 Safari/537.36")
)

func (u UserAgent) String() string {
	return string(u)
}

func (u *UserAgent) UnmarshalText(text []byte) error {
	agent := string(text)
	switch agent {
	case ":firefox:":
		*u = Firefox88
	case ":safari:":
		*u = Safari537
	default:
		*u = UserAgent(agent)
	}
	return nil
}
