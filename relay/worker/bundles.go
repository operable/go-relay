package worker

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/operable/go-relay/relay"
	"github.com/operable/go-relay/relay/config"
	"github.com/operable/go-relay/relay/engines"
	"github.com/operable/go-relay/relay/messages"
	"golang.org/x/net/context"
)

func updateBundles(ctx context.Context, listBundles *messages.ListBundlesResponseEnvelope) {
	incoming := ctx.Value("incoming").(*relay.Incoming)
	for _, bundle := range listBundles.Bundles {
		bundleConfig := bundle.ConfigFile
		if bundleConfig.IsDocker() {
			log.Infof("Downloading Docker image %s for bundle %s.", bundleConfig.Docker.PrettyImageName(),
				bundleConfig.Name)
			err := fetchImage(incoming.Relay.Config, &bundleConfig)
			if err != nil {
				log.Errorf("Failed to download Docker image %s for bundle %s: %s.", bundleConfig.Docker.PrettyImageName(),
					bundleConfig.Name, err)
				continue
			}
			log.Infof("Downloaded Docker image %s for bundle %s.", bundleConfig.Docker.PrettyImageName(),
				bundleConfig.Name)
		}
		incoming.Relay.StoreBundle(&bundleConfig)
	}
	incoming.Relay.FinishedUpdateBundles()
}

func fetchImage(relayConfig *config.Config, bundle *config.Bundle) error {
	docker, err := engines.NewDockerEngine(*relayConfig)
	if err != nil {
		return err
	}
	if docker == nil {
		return fmt.Errorf("Docker engine is disabled.")
	}
	isAvail, err := docker.IsAvailable(bundle.Docker.Image, bundle.Docker.Tag)
	if err != nil {
		return err
	}
	if isAvail == false {
		return fmt.Errorf("Not found")
	}
	bundle.Docker.ID, err = docker.IDForName(bundle.Docker.Image)
	if err != nil {
		return err
	}
	return nil
}
