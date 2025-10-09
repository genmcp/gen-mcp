package logging

import "encoding/json"

func (lc *LoggingConfig) UnmarshalJSON(data []byte) error {
	type Doppleganger LoggingConfig

	tmp := struct {
		*Doppleganger
		EnableMcpLogs *bool `json:"enableMcpLogs"`
	}{
		Doppleganger: (*Doppleganger)(lc),
	}

	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}

	lc.enableMcpLogs = tmp.EnableMcpLogs

	return nil
}
