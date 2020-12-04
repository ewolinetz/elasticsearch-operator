package k8shandler

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"github.com/ViaQ/logerr/log"
	v1 "github.com/openshift/elasticsearch-operator/apis/logging/v1"
	"github.com/openshift/elasticsearch-operator/internal/alertmanager"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
)

const max_node_count = 5

func (er *ElasticsearchRequest) ReconfigureCR() error {

	httpClient := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
				DualStack: true,
			}).DialContext,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	amClient := alertmanager.NewClient("https://alertmanager-main-openshift-monitoring.apps.<user>.devcluster.openshift.com",
		httpClient, "<kube-admin token>")
	alerts, err := amClient.Alerts()

	if err != nil {
		return err
	}

	if alerts == nil {
		log.Info("Alerts are nil")
		return nil
	}

	// 0. This hackathon demo requires that masters are not also data nodes
	// 1. evaluate alerts and adjust the cr we know about
	// 2. compare the adjusted cr to what is currently out there -- it may not be the same
	// 3. if different, patch it *
	// * there a lot of nuances that can happen but for the sake of this demo its going to be a primitive change and assume only the node counts need to change

	// queue up several different changes at once
	cluster := er.cluster

	log.Info("Received alerts", "alerts", alerts)

	// scale up data/ingest
	if alerts.HeapHigh || alerts.DiskAvailabilityLow || alerts.LowWatermark {
		for index, node := range cluster.Spec.Nodes {
			if isDataNode(node) {
				if cluster.Spec.Nodes[index].NodeCount < max_node_count {
					log.Info("scaling up data nodes!", "alerts", alerts)
					cluster.Spec.Nodes[index].NodeCount += 1
				}
			}
		}
	}

	// scale up master based on number of data nodes
	dataCount := getDataCount(cluster)
	masterCount := getMasterCount(cluster)

	if dataCount >= 3 && masterCount < 3 {
		for index, node := range cluster.Spec.Nodes {
			if isMasterNode(node) {
				log.Info("Scaling up master nodes!")
				cluster.Spec.Nodes[index].NodeCount = 3
			}
		}
	}

	currentCR := &v1.Elasticsearch{}
	objectKey := types.NamespacedName{
		Name:      cluster.Name,
		Namespace: cluster.Namespace,
	}

	if err := er.client.Get(context.TODO(), objectKey, currentCR); err != nil {
		return err
	}

	// check if the master and data node counts already match
	different := false
	for _, currentNode := range currentCR.Spec.Nodes {
		for _, desiredNode := range cluster.Spec.Nodes {
			if (isMasterNode(currentNode) && isMasterNode(desiredNode)) ||
				(isDataNode(currentNode) && isDataNode(desiredNode)) {

				if currentNode.NodeCount < desiredNode.NodeCount {
					different = true
				}
			}
		}
	}

	if different {
		// update the CR
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			if err := er.client.Get(context.TODO(), objectKey, currentCR); err != nil {
				return err
			}

			for currentIndex, currentNode := range currentCR.Spec.Nodes {
				for _, desiredNode := range cluster.Spec.Nodes {
					if (isMasterNode(currentNode) && isMasterNode(desiredNode)) ||
						(isDataNode(currentNode) && isDataNode(desiredNode)) {

						if currentNode.NodeCount < desiredNode.NodeCount {
							currentCR.Spec.Nodes[currentIndex].NodeCount = desiredNode.NodeCount
						}
					}
				}
			}

			if err := er.client.Update(context.TODO(), currentCR); err != nil {
				return err
			}
			return nil
		})
		return err
	}

	return nil
}
