package elasticsearch

import (
	"fmt"
	"net/http"
	"strconv"
)

// This will idempotently update the index templates and update indices' replica count
func (ec *esClient) UpdateReplicaCount(replicaCount int32) error {
	if ok, _ := ec.updateAllIndexTemplateReplicas(replicaCount); ok {
		if _, err := ec.updateAllIndexReplicas(replicaCount); err != nil {
			return err
		}
	}
	return nil
}

func (ec *esClient) updateAllIndexReplicas(replicaCount int32) (bool, error) {
	indexHealth, _ := ec.GetIndexReplicaCounts()

	// get list of indices and call updateIndexReplicas for each one
	for index, health := range indexHealth {
		if healthMap, ok := health.(map[string]interface{}); ok {
			// only update replicas for indices that don't have same replica count
			if numberOfReplicas := parseString("settings.index.number_of_replicas", healthMap); numberOfReplicas != "" {
				currentReplicas, err := strconv.ParseInt(numberOfReplicas, 10, 32)
				if err != nil {
					return false, err
				}

				if int32(currentReplicas) != replicaCount {
					// best effort initially?
					if ack, err := ec.updateIndexReplicas(index, replicaCount); err != nil {
						return ack, err
					}
				}
			}
		} else {
			return false, ec.errorCtx().New("unable to evaluate the number of replicas for index",
				"index", index,
				"health", health,
			)
		}
	}

	return true, nil
}

func (ec *esClient) GetIndexReplicaCounts() (map[string]interface{}, error) {

	es := ec.client
	res, err := es.Indices.GetSettings(es.Indices.GetSettings.WithName("index.number_of_replicas"), es.Indices.GetSettings.WithPretty())

	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() || res.StatusCode != http.StatusOK {
		return nil, ec.errorCtx().New("failed to get replicas",
			"response_error", res.String,
			"response_status", res.StatusCode,
			"response_body", res.Body)
	}

	body, _ := ioutil.ReadAll(res.Body)
	jsonStr := string(body)
	m := make(map[string]interface{})

	if jsonStr == "" {
		return m, nil
	}

	err = json.Unmarshal(body, &m)

	return payload.ResponseBody, payload.Error
}

func (ec *esClient) updateIndexReplicas(index string, replicaCount int32) (bool, error) {

	es := ec.client
	settings := fmt.Sprintf("{%q:\"%d\"}}", "index.number_of_replicas", replicaCount)
	body := ioutil.NopCloser(bytes.NewBufferString(settings))
	res, err := es.Indices.PutSettings(body, es.Indices.PutSettings.WithIndex(index), es.Indices.PutSettings.WithPretty())

	if err != nil {
		return false, err
	}
	defer res.Body.Close()
	acknowledged := false
	if res.IsError() || res.StatusCode != http.StatusOK {
		return false, ec.errorCtx().New("failed to update Index replicas",
			"response_error", res.String,
			"response_status", res.StatusCode,
			"response_body", res.Body)
	} else {
		acknowledged = true
	}

	if !acknowledged {
		return false, fmt.Errorf("failed to update Index replicas %s  cluster in %s namespace", ec.cluster, ec.namespace)
	}
	return payload.StatusCode == 200 && acknowledged, payload.Error
}
