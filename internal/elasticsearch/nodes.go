package elasticsearch

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/inhies/go-bytesize"
	"k8s.io/apimachinery/pkg/api/resource"
)

func (ec *esClient) GetNodeDiskUsage(nodeName string) (string, float64, error) {
	payload := &EsRequest{
		Method: http.MethodGet,
		URI:    "_nodes/stats/fs",
	}

	ec.fnSendEsRequest(ec.cluster, ec.namespace, payload, ec.k8sClient)

	usage := ""
	percentUsage := float64(-1)

	if payload, ok := payload.ResponseBody["nodes"].(map[string]interface{}); ok {
		for _, stats := range payload {
			// ignore the key name here, it is the node UUID
			if parseString("name", stats.(map[string]interface{})) == nodeName {
				total := parseFloat64("fs.total.total_in_bytes", stats.(map[string]interface{}))
				available := parseFloat64("fs.total.available_in_bytes", stats.(map[string]interface{}))

				percentUsage = (total - available) / total * 100.00
				usage = strings.TrimSuffix(fmt.Sprintf("%s", bytesize.New(total)-bytesize.New(available)), "B")

				break
			}
		}
	}

	return usage, percentUsage, payload.Error
}

func (ec *esClient) NodesExceedingUsage() bool {

	var DiskWatermarkLowPct *float64
	var DiskWatermarkLowAbs *resource.Quantity

	low, _, _ := ec.GetDiskWatermarks()

	switch low.(type) {
	case float64:
		value := low.(float64)
		DiskWatermarkLowPct = &value
		DiskWatermarkLowAbs = nil
	case string:
		value, _ := resource.ParseQuantity(strings.ToUpper(low.(string)))
		DiskWatermarkLowAbs = &value
		DiskWatermarkLowPct = nil
	}

	payload := &EsRequest{
		Method: http.MethodGet,
		URI:    "_nodes/stats/fs",
	}

	ec.fnSendEsRequest(ec.cluster, ec.namespace, payload, ec.k8sClient)

	usage := ""
	percentUsage := float64(-1)

	if payload, ok := payload.ResponseBody["nodes"].(map[string]interface{}); ok {
		for _, stats := range payload {

			total := parseFloat64("fs,total,total_in_bytes", stats.(map[string]interface{}))
			available := parseFloat64("fs,total,available_in_bytes", stats.(map[string]interface{}))

			percentUsage = (total - available) / total * 100.00
			usage = strings.TrimSuffix(fmt.Sprintf("%s", bytesize.New(total)-bytesize.New(available)), "B")

			if exceedsWatermarks(usage, percentUsage, DiskWatermarkLowAbs, DiskWatermarkLowPct) {
				return true
			}
		}
	}

	return false
}

func exceedsWatermarks(usage string, percent float64, watermarkUsage *resource.Quantity, watermarkPercent *float64) bool {
	if usage == "" || percent < float64(0) {
		return false
	}

	quantity, err := resource.ParseQuantity(usage)
	if err != nil {
		return false
	}

	// if quantity is > watermarkUsage and is used
	if watermarkUsage != nil && quantity.Cmp(*watermarkUsage) == 1 {
		return true
	}

	if watermarkPercent != nil && percent > *watermarkPercent {
		return true
	}

	return false
}
