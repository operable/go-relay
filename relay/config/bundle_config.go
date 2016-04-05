package config

import (
	"encoding/json"
	"github.com/asaskevich/govalidator"
)

// Bundle represents a command bundle's complete configuration
type Bundle struct {
	BundleVersion int                        `json:"cog_bundle_version" valid:"required"`
	Name          string                     `json:"name" valid:"required"`
	Version       string                     `json:"version" valid:"semver,required"`
	Permissions   []string                   `json:"permissions"`
	Docker        *DockerImage               `json:"docker"`
	Commands      map[string]*BundleCommand  `json:"commands"`
	Templates     map[string]*BundleTemplate `json:"templates"`
}

// DockerImage identifies the bundle's image name and version
type DockerImage struct {
	Image string `json:"image" valid:"required"`
	Tag   string `json:"tag"`
}

// BundleCommand identifies a command within a bundle
type BundleCommand struct {
	Name       string
	Executable string                          `json:"executable" valid:"required"`
	Options    map[string]*BundleCommandOption `json:"options"`
	Rules      []string                        `json:"rules"`
}

// BundleCommandOption is a description of a command's option
type BundleCommandOption struct {
	Name        string
	Type        string `json:"type"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
	ShortFlag   string `json:"short_flag"`
}

type BundleTemplate struct {
	Name     string
	Provider string `json:"provider" valid:"required"`
	Contents string `json:"contents" valid:"required"`
}

func validateBundleConfig(bundle *Bundle) error {
	_, err := govalidator.ValidateStruct(bundle)
	if err == nil && bundle.Docker != nil {
		_, err = govalidator.ValidateStruct(bundle.Docker)
	}
	return err
}

func ParseBundleConfig(data []byte) (*Bundle, error) {
	result := &Bundle{}
	if err := json.Unmarshal(data, result); err != nil {
		return nil, err
	}

	// Remove Docker struct if no values exist
	if result.Docker != nil && (result.Docker.Tag == "" &&
		result.Docker.Image == "") {
		result.Docker = nil
	}
	if err := validateBundleConfig(result); err != nil {
		return nil, err
	}
	for name, command := range result.Commands {
		for optname, option := range command.Options {
			option.Name = optname
		}
		command.Name = name
	}
	for name, template := range result.Templates {
		template.Name = name
	}
	return result, nil
}
