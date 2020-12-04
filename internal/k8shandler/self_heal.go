package k8shandler

import (
	"context"

	v1 "github.com/openshift/elasticsearch-operator/apis/logging/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
)

const max_node_count = 5

func (er *ElasticsearchRequest) ReconfigureCR() error {

	var am AlertManager
	alerts := am.Alerts()

	if alerts == nil {
		return nil
	}

	// 0. This hackathon demo requires that masters are not also data nodes
	// 1. evaluate alerts and adjust the cr we know about
	// 2. compare the adjusted cr to what is currently out there -- it may not be the same
	// 3. if different, patch it *
	// * there a lot of nuances that can happen but for the sake of this demo its going to be a primitive change and assume only the node counts need to change

	// queue up several different changes at once
	cluster := er.cluster

	// scale up data/ingest
	if alerts.heap_high || alerts.disk_usage_low || alerts.low_watermark {
		for index, node := range cluster.Spec.Nodes {
			if isDataNode(node) {
				cluster.Spec.Nodes[index].NodeCount += 1
			}
		}
	}

	// scale up master based on number of data nodes
	dataCount := getDataCount(cluster)
	masterCount := getMasterCount(cluster)

	if dataCount >= 3 && masterCount < 3 {
		for index, node := range cluster.Spec.Nodes {
			if isMasterNode(node) {
				cluster.Spec.Nodes[index].NodeCount = 3
			}
		}
	}

	var currentCR *v1.Elasticsearch
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
