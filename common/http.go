package common

import (
	"encoding/json"
	"io"
)

func UnmarshalBody(r io.Reader, v any) error {
	body, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, v); err != nil {
		return err
	}
	return nil
}
