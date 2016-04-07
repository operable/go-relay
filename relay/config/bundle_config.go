package config

import (
	"encoding/json"
	"fmt"
	"github.com/asaskevich/govalidator"
	"time"
)

// Bundle represents a command bundle's complete configuration
type Bundle struct {
	BundleVersion int                        `json:"cog_bundle_version" valid:"required"`
	Name          string                     `json:"name" valid:"required"`
	Version       string                     `json:"version" valid:"semver,required"`
	Permissions   []string                   `json:"permissions"`
	Docker        *DockerImage               `json:"docker" valid:"-"`
	Commands      map[string]*BundleCommand  `json:"commands" valid:"-"`
	Templates     map[string]*BundleTemplate `json:"templates" valid:"-"`
}

// DockerImage identifies the bundle's image name and version
type DockerImage struct {
	Image       string    `json:"image" valid:"notempty,required"`
	Tag         string    `json:"tag" valid:"-"`
	ID          string    `valid:"-"`
	RetrievedAt time.Time `valid:"-"`
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

// BundleTemplate is an output template
type BundleTemplate struct {
	Name    string
	Slack   string `json:"slack,omitempty" valid:"-"`
	HipChat string `json:"hipchat,omitempty" valid:"-"`
	IRC     string `json:"irc,omitempty" valid:"-"`
}

// IsDocker returns true if the bundle contains a Docker stanza
func (b *Bundle) IsDocker() bool {
	return b.Docker != nil
}

// PrettyImageName returns a prettified version of a Docker image
// include repository, name, and tag
func (di *DockerImage) PrettyImageName() string {
	return fmt.Sprintf("%s::%s", di.Image, di.Tag)
}

func validateBundleConfig(bundle *Bundle) error {
	_, err := govalidator.ValidateStruct(bundle)
	if err == nil && bundle.IsDocker() {
		_, err = govalidator.ValidateStruct(bundle.Docker)
	}
	return err
}

// ParseBundleConfig parses raw bundle configs sent by
// Cog
func ParseBundleConfig(data []byte) (*Bundle, error) {
	govalidator.TagMap["notempty"] = govalidator.Validator(func(str string) bool {
		return str != ""
	})
	result := &Bundle{}
	if err := json.Unmarshal(data, result); err != nil {
		return nil, err
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
