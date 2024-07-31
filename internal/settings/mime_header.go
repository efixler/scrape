package settings

import (
	"encoding/json"
	"net/textproto"
)

type MIMEHeader map[string]string

func (ch MIMEHeader) MarshalJSON() ([]byte, error) {
	mm := make(map[string]string, len(ch))
	for k, v := range ch {
		mm[textproto.CanonicalMIMEHeaderKey(k)] = v
	}
	return json.Marshal(mm)
}
