package minestom

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type PushManifest struct {
	Name      string
	FlavorKey string
	Runtime   Runtime
	Full      map[string]any
}

type Runtime struct {
	Type       string   `yaml:"type" json:"type"`
	PublicType string   `yaml:"-" json:"publicType,omitempty"`
	BaseImage  string   `yaml:"baseImage" json:"baseImage"`
	Build      Build    `yaml:"build" json:"-"`
	Modules    []Module `yaml:"modules" json:"-"`
}

type Build struct {
	Task     string `yaml:"task"`
	Artifact string `yaml:"artifact"`
}

type Module struct {
	ID      string `yaml:"id"`
	Variant string `yaml:"variant"`
	Source  string `yaml:"source"`
}

type runtimeValidationError struct {
	message string
}

func (e *runtimeValidationError) Error() string {
	return e.message
}

func IsRuntimeValidationError(err error) bool {
	var validationErr *runtimeValidationError
	return errors.As(err, &validationErr)
}

func (m PushManifest) IsMinestomServer() bool {
	return m.Runtime.Type == "minestom-server"
}

func LoadPushManifest(path, flavor string) (*PushManifest, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc struct {
		Name      string             `yaml:"name"`
		Type      string             `yaml:"type"`
		BaseImage string             `yaml:"baseImage"`
		Build     Build              `yaml:"build"`
		Modules   []Module           `yaml:"modules"`
		Flavors   map[string]Runtime `yaml:"flavors"`
	}
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	var full map[string]any
	if err := yaml.Unmarshal(raw, &full); err != nil {
		return nil, err
	}
	if len(doc.Flavors) > 0 {
		key := strings.TrimSpace(flavor)
		if key == "" {
			return nil, fmt.Errorf("grounds.yaml: flavor selection required (available=%s)", availableFlavorKeys(doc.Flavors))
		}
		selected, ok := doc.Flavors[key]
		if !ok {
			return nil, fmt.Errorf("grounds.yaml: unknown flavor %q (available=%s)", key, availableFlavorKeys(doc.Flavors))
		}
		if selected.Type == "minestom" {
			selected.Type = "minestom-server"
		}
		selected.PublicType = publicTypeFor(selected.Type)
		if err := validateRuntime(fmt.Sprintf("minestom flavor %q", key), selected); err != nil {
			return nil, err
		}
		return &PushManifest{Name: strings.TrimSpace(doc.Name), FlavorKey: key, Runtime: selected, Full: full}, nil
	}
	runtime := Runtime{
		Type:      strings.TrimSpace(doc.Type),
		BaseImage: strings.TrimSpace(doc.BaseImage),
		Build:     doc.Build,
		Modules:   doc.Modules,
	}
	if runtime.Type == "minestom" {
		runtime.Type = "minestom-server"
	}
	runtime.PublicType = publicTypeFor(runtime.Type)
	if err := validateRuntime("minestom runtime", runtime); err != nil {
		return nil, err
	}
	return &PushManifest{Name: strings.TrimSpace(doc.Name), FlavorKey: runtime.PublicType, Runtime: runtime, Full: full}, nil
}

func validateRuntime(subject string, runtime Runtime) error {
	if runtime.Type != "minestom-server" {
		return nil
	}
	var missing []string
	if strings.TrimSpace(runtime.Build.Task) == "" {
		missing = append(missing, "build.task")
	}
	if strings.TrimSpace(runtime.Build.Artifact) == "" {
		missing = append(missing, "build.artifact")
	}
	if len(runtime.Modules) == 0 {
		missing = append(missing, "modules")
	}
	if len(missing) > 0 {
		return &runtimeValidationError{message: fmt.Sprintf("grounds.yaml: %s missing %s", subject, strings.Join(missing, ", "))}
	}
	for i, module := range runtime.Modules {
		if strings.TrimSpace(module.ID) == "" {
			return &runtimeValidationError{message: fmt.Sprintf("grounds.yaml: minestom module at index %d missing id", i)}
		}
		if strings.TrimSpace(module.Source) == "" {
			return &runtimeValidationError{message: fmt.Sprintf("grounds.yaml: minestom module %q missing source", module.ID)}
		}
	}
	return nil
}

func publicTypeFor(runtimeType string) string {
	if runtimeType == "minestom-server" {
		return "minestom"
	}
	return runtimeType
}

func availableFlavorKeys(flavors map[string]Runtime) string {
	keys := make([]string, 0, len(flavors))
	for key := range flavors {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return strings.Join(keys, ", ")
}
