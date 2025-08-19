package loader

type ConfigLoader interface {
	Load() (map[string]string, error)
}
