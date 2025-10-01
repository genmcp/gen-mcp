package mcpfile

import (
	"encoding/json"
)

func (t *Tool) MarshalJSON() ([]byte, error) {
	type Doppleganger Tool

	tmp := &struct {
		Invocation map[string]json.RawMessage `json:"invocation"`
		*Doppleganger
	}{
		Invocation:   make(map[string]json.RawMessage),
		Doppleganger: (*Doppleganger)(t),
	}

	tmp.Invocation[t.InvocationType] = t.InvocationData

	return json.Marshal(tmp)
}

func (p *Prompt) MarshalJSON() ([]byte, error) {
	type Doppleganger Prompt

	tmp := &struct {
		Invocation map[string]json.RawMessage `json:"invocation"`
		*Doppleganger
	}{
		Invocation:   make(map[string]json.RawMessage),
		Doppleganger: (*Doppleganger)(p),
	}

	tmp.Invocation[p.InvocationType] = p.InvocationData

	return json.Marshal(tmp)
}
