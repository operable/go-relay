package worker

import (
	log "github.com/Sirupsen/logrus"
	"github.com/operable/go-relay/relay/messages"
	"golang.org/x/net/context"
)

func UpdateBundles(ctx context.Context, listBundles *messages.ListBundlesResponseEnvelope) {
	log.Infof("Processed %d bundle assignments", len(listBundles.Bundles))
}
