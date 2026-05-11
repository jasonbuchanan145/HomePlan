package house

import (
	"encoding/json"
	"errors"
)

var ErrInvalidState = errors.New("invalid house state")
var ErrNotFound = errors.New("not found")

func ValidateState(raw json.RawMessage) error {
	var state struct {
		SchemaVersion int                    `json:"schemaVersion"`
		ID            string                 `json:"id"`
		Name          string                 `json:"name"`
		Floors        map[string]interface{} `json:"floors"`
	}
	if err := json.Unmarshal(raw, &state); err != nil {
		return err
	}
	if state.SchemaVersion != 1 || state.ID == "" || state.Name == "" || len(state.Floors) == 0 {
		return ErrInvalidState
	}
	return nil
}
