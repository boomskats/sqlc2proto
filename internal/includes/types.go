package includes

// IncludesFile represents the structure of the includes YAML file
type IncludesFile struct {
	Models  []string `yaml:"models"`
	Queries []string `yaml:"queries"`
}

// NewEmptyIncludesFile creates a new empty includes file
func NewEmptyIncludesFile() IncludesFile {
	return IncludesFile{
		Models:  []string{},
		Queries: []string{},
	}
}
