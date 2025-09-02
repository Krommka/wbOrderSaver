package dotEnvLoader

import (
	"github.com/joho/godotenv"
	"os"
	"strings"
)

type DotEnvLoader struct{}

func (l DotEnvLoader) Load() (map[string]string, error) {
	_ = godotenv.Load()
	envs := make(map[string]string)
	for _, env := range os.Environ() {
		key, val, _ := strings.Cut(env, "=")
		envs[key] = val
	}
	return envs, nil
}
