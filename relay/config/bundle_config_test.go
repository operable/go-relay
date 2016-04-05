package config

import (
	"fmt"
	"testing"
)

const (
	DockerBundle = `{
  "cog_bundle_version": 2,
  "name": "test_bundle",
  "version": "0.1.0",
  "permissions": [
	"test_bundle:date",
	"test_bundle:time"
  ],
  "docker": {
	"image": "operable-bundle/test_bundle",
	"tag": "v0.1.0"
  },
  "commands": {
	"date": {
	  "executable": "/usr/local/bin/date",
	  "options": {
		"option1": {
		  "type": "string",
		  "description": "An option",
		  "required": false,
		  "short_flag": "o"
		}
	  },
	  "rules": [
		"when command is test_bundle:date must have test_bundle:date"
	  ]
	},
	"time": {
	  "executable": "/usr/local/bin/time",
	  "rules": [
		"when command is test_bundle:time must have test_bundle:time"
	  ]
	}
  },
  "templates": {
	"time": {
	  "provider": "slack",
	  "contents": "{{time}}"
	},
	"date": {
	  "provider": "slack",
	  "contents": "{{date}}"
	}
  }
}`
	MissingDockerImageName = `{
  "cog_bundle_version": 2,
  "name": "test_bundle",
  "version": "0.1.0",
  "permissions": [
	"test_bundle:date",
	"test_bundle:time"
  ],
  "docker": {
	"tag": "v0.1.0"
  },
  "commands": {
	"date": {
	  "executable": "/usr/local/bin/date",
	  "options": {
		"option1": {
		  "type": "string",
		  "description": "An option",
		  "required": false,
		  "short_flag": "o"
		}
	  },
	  "rules": [
		"when command is test_bundle:date must have test_bundle:date"
	  ]
	},
	"time": {
	  "executable": "/usr/local/bin/time",
	  "rules": [
		"when command is test_bundle:time must have test_bundle:time"
	  ]
	}
  },
  "templates": {
	"time": {
	  "provider": "slack",
	  "contents": "{{time}}"
	},
	"date": {
	  "provider": "slack",
	  "contents": "{{date}}"
	}
  }
}`
	NonDockerBundle = `{
  "cog_bundle_version": 2,
  "name": "test_bundle",
  "version": "0.1.0",
  "permissions": [
	"test_bundle:date",
	"test_bundle:time"
  ],
  "commands": {
	"date": {
	  "executable": "/usr/local/bin/date",
	  "options": {
		"option1": {
		  "type": "string",
		  "description": "An option",
		  "required": false,
		  "short_flag": "o"
		}
	  },
	  "rules": [
		"when command is test_bundle:date must have test_bundle:date"
	  ]
	},
	"time": {
	  "executable": "/usr/local/bin/time",
	  "rules": [
		"when command is test_bundle:time must have test_bundle:time"
	  ]
	}
  },
  "templates": {
	"time": {
	  "provider": "slack",
	  "contents": "{{time}}"
	},
	"date": {
	  "provider": "slack",
	  "contents": "{{date}}"
	}
  }
}`
)

func TestParseDockerBundle(t *testing.T) {
	config, err := ParseBundleConfig([]byte(DockerBundle))
	if err != nil {
		t.Fatal(err)
	}
	if len(config.Commands) != 2 {
		t.Errorf("Expected 2 commands. Found %d.", len(config.Commands))
	}
	checkNames(config, t)
}

func TestParseMissingDockerImageName(t *testing.T) {
	config, err := ParseBundleConfig([]byte(MissingDockerImageName))
	if err == nil {
		t.Errorf("Expected missing Docker image name to trigger parsing error.")
	}
	if fmt.Sprintf("%s", err) != "Image: non zero value required;" {
		t.Errorf("Expected error message to reference missing Image field")
	}
	if config != nil {
		t.Errorf("Expected bad config to return nil: %+v.", config.Docker)
	}
}

func TestParseNonDockerBundle(t *testing.T) {
	config, err := ParseBundleConfig([]byte(NonDockerBundle))
	if err != nil {
		t.Fatal(err)
	}
	if config.Docker != nil {
		t.Errorf("Expected Docker stanza to be nil: %+v.", config.Docker)
	}
	checkNames(config, t)
}

func checkNames(config *Bundle, t *testing.T) {
	for cname, command := range config.Commands {
		if command.Name == "" {
			t.Errorf("Command name for key '%s' not set.", cname)
		}
		for oname, option := range command.Options {
			if option.Name == "" {
				t.Errorf("Option name for key '%s' not set.", oname)
			}
		}
	}
	for tname, template := range config.Templates {
		if template.Name == "" {
			t.Errorf("Template name for key '%s' not set.", tname)
		}
	}

}
