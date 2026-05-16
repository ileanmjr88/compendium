package registry

import "encoding/json"

// The entire index.json
type Index struct {
	Version   string                `json:"version"`
	UpdatedAt string                `json:"updated_at"`
	Languages map[string]IndexEntry `json:"languages"`
	Tools     map[string]IndexEntry `json:"tools"`
}

type IndexEntry struct {
	Latest   string `json:"latest"`
	Versions int    `json:"versions"`
	File     string `json:"file"`
}

type ToolFile map[string]VersionEntry

type VersionEntry struct {
	Depends   map[string]string
	Platforms PlatformMap
}

type PlatformMap map[string]ArchMap

type ArchMap map[string]Artifact

type Artifact struct {
	URL      string `json:"url"`
	Checksum string `json:"checksum"`
	Size     int64  `json:"size"`
	Strip    int    `json:"strip"`
}

func (v *VersionEntry) UnmarshalJSON(data []byte) error {
	raw := map[string]json.RawMessage{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	v.Platforms = PlatformMap{}
	for k, val := range raw {
		if k == "depends" {
			if err := json.Unmarshal(val, &v.Depends); err != nil {
				return err
			}
			continue
		}
		var am ArchMap
		if err := json.Unmarshal(val, &am); err != nil {
			return err
		}
		v.Platforms[k] = am
	}
	return nil
}
