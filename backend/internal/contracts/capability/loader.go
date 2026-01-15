package capability

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	yaml "gopkg.in/yaml.v3"
)

// Record couples a descriptor with its source file.
type Record struct {
	Descriptor Descriptor
	Source     string
}

// Catalog indexes capability descriptors by ID.
type Catalog struct {
	records map[string]Record
}

// LoadCatalog walks the provided directory and decodes *.yaml capability descriptors.
func LoadCatalog(dir string) (*Catalog, error) {
	if dir == "" {
		return nil, errors.New("capability directory must be provided")
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("capability directory does not exist: %w", err)
		}
		return nil, fmt.Errorf("read capability directory: %w", err)
	}

	catalog := &Catalog{records: make(map[string]Record)}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !isYAML(entry.Name()) {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read capability file %s: %w", path, err)
		}

		var descriptor Descriptor
		if err := yaml.Unmarshal(data, &descriptor); err != nil {
			return nil, fmt.Errorf("decode capability %s: %w", path, err)
		}

		if strings.TrimSpace(descriptor.ID) == "" {
			return nil, fmt.Errorf("capability file %s is missing id", path)
		}
		if _, exists := catalog.records[descriptor.ID]; exists {
			return nil, fmt.Errorf("duplicate capability id %s encountered in %s", descriptor.ID, path)
		}

		catalog.records[descriptor.ID] = Record{
			Descriptor: descriptor,
			Source:     path,
		}
	}

	return catalog, nil
}

// Get returns the descriptor for the given capability ID.
func (c *Catalog) Get(id string) (Record, bool) {
	if c == nil {
		return Record{}, false
	}
	rec, ok := c.records[id]
	return rec, ok
}

// List returns the descriptors in alphabetical order of ID.
func (c *Catalog) List() []Record {
	if c == nil {
		return nil
	}
	out := make([]Record, 0, len(c.records))
	for _, rec := range c.records {
		out = append(out, rec)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Descriptor.ID < out[j].Descriptor.ID
	})
	return out
}

func isYAML(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasSuffix(lower, ".yaml") || strings.HasSuffix(lower, ".yml")
}
