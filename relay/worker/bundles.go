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

func isBundleNew(name string, existing []string) bool {
	for _, bundleName := range existing {
		if name == bundleName {
			return false
		}
	}
	return true
}

func updateBundles(ctx context.Context, listBundles *messages.ListBundlesResponseEnvelope) {
	incoming := ctx.Value("incoming").(*relay.Incoming)
	existingBundleNames := incoming.Relay.BundleNames()
	bundles := make(map[string]*config.Bundle)
	images := []string{}
	for _, bundle := range listBundles.Bundles {
		bundleConfig := bundle.ConfigFile
		if bundleConfig.IsDocker() {
			newImages, err := fetchImage(incoming.Relay.Config, &bundleConfig, images)
			if err != nil {
				log.Errorf("Docker image %s for bundle %s is not available: %s.", bundleConfig.Docker.PrettyImageName(),
					bundleConfig.Name, err)
			} else if isBundleNew(bundleConfig.Name, existingBundleNames) {
				log.Infof("Docker image %s for bundle %s is available.", bundleConfig.Docker.PrettyImageName(),
					bundleConfig.Name)
			}
			images = newImages
		} else if incoming.Relay.Config.NativeEnabled() == false {
			log.Errorf("Native execution engine disabled. Bundle %s ignored.", bundleConfig.Name)
			continue
		}
		bundles[bundleConfig.Name] = &bundleConfig
	}
	incoming.Relay.UpdateBundleList(bundles)
	incoming.Relay.FinishedUpdateBundles()
}

func fetchImage(relayConfig *config.Config, bundle *config.Bundle, fetched []string) ([]string, error) {
	docker, err := engines.NewDockerEngine(*relayConfig)
	if err != nil {
		return fetched, err
	}
	if docker == nil {
		return fetched, fmt.Errorf("Docker engine is disabled.")
	}
	prettyName := bundle.Docker.PrettyImageName()
	for _, v := range fetched {
		if v == prettyName {
			return fetched, nil
		}
	}
	isAvail, err := docker.IsAvailable(bundle.Docker.Image, bundle.Docker.Tag)
	if err != nil {
		return fetched, err
	}
	if isAvail == false {
		return fetched, fmt.Errorf("Not found")
	}
	fetched = append(fetched, bundle.Docker.PrettyImageName())
	if err != nil {
		return fetched, err
	}
	return fetched, nil
}
