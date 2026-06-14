package preset

import (
	"encoding/json"
	"os"
)

func Load(path string) (*Preset, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var p Preset

	if err := json.Unmarshal(data, &p); err != nil {
		return nil, err
	}

	return &p, nil
}
