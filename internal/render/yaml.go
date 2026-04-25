package render

import (
	"io"

	"gopkg.in/yaml.v3"
)

func YAML(w io.Writer, v any) error { return yaml.NewEncoder(w).Encode(v) }
