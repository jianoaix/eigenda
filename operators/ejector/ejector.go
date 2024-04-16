package ejector

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"sync"

	"cloud.google.com/go/logging"
	"github.com/Layr-Labs/eigenda/core"
	"github.com/Layr-Labs/eigenda/disperser/dataapi"
)

// The caller should ensure "stakeShare" is in range (0, 1].
func stakeShareToSLA(stakeShare float64) float64 {
	switch stakeShare {
	case stakeShare > 0.1:
		return 0.975
	case stakeShare > 0.05:
		return 0.95
	default:
		return 0.9
	}
}

// operatorPerfScore scores an operator based on its stake share and nonsigning rate. The
// performance score will be in range [0, 1], with higher score indicating better performance.
func operatorPerfScore(stakeShare float64, nonsigningRate float64) float64 {
	if nonsigningRate == 0 {
		return 1.0
	}
	sla := stakeShareToSLA(stakeShare)
	perf := (1 - sla) / nonsigningRate
	return perf / (1.0 + perf)
}

func computePerfScore(metric *OperatorNonsigningPercentageMetrics) float64 {
	return operatorPerfScore(metric.StakePercentage, metric.Percentage/100.0)
}

type Ejector struct {
	Logger            logging.Logger
	Transactor        core.Transactor
	nonsigningRateUri string
	Metrics           *Metrics

	// For serializing the ejection requests.
	mu sync.Mutex
}

func NewEjector(config *Config, logger logging.Logger, tx core.Transactor, metric *Metrics) *Ejector {
	return &Ejector{
		Logger:            logger.With("component", "Ejector"),
		Transactor:        tx,
		nonsigningRateUri: fmt.Sprintf("%s%s", config.DataApiHostName, config.NonsigningRateApiPath),
		Metrics:           metrics,
	}
}

func (e *Ejector) eject(ctx context.Context, lookbackWindow, endTime int64) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	nonsigners, err := e.getNonsigners(lookbackWindow, endTime)
	if err != nil {
		return
	}

	// Rank the operators for each quorum by the operator performance score.
	// The operators with lower perf score will get ejected with priority in case of
	// rate limiting.
	sort.Slice(nonsigners, func(i, j int) bool {
		if nonsigners[i].QuorumId == nonsigners[j].QuorumId {
			return computePerfScore(nonsigners[i]) < computePerfScore(nonsigners[j])
		}
		return nonsigners[i].QuorumId < nonsigners[j].QuorumId
	})

	operatorsByQuorum := convertOperators(nonsigners)

	// TODO: execute the ejection

	return nil
}

func (e *Ejection) getNonsigners(lookbackWindow, endTime int64) ([]*dataapi.OperatorNonsigningPercentageMetrics, error) {
	uri := fmt.Sprintf("%s?end=%d&interval=%d", e.nonsigningRateUri, endTime, lookbackWindow)
	response, err := http.Get(uri)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.body)
	if err != nil {
		return nil, err
	}

	var data dataapi.OperatorsNonsigningPercentage
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	nonsigners := make([]*OperatorNonsigningPercentageMetrics)
	for _, metric := range data.Data {
		// Collect only the nonsigners who violate the SLA.
		if metric.Percentage/100.0 > 1-stakeShareToSLA(metric.StakePercentage) {
			nonsigners = append(nonsigners, metric)
		}
	}

	return nonsigners, nil
}

func (e *Ejection) convertOperators(nonsigners []*dataapi.OperatorNonsigningPercentageMetrics) [][]string {
	var maxQuorumId uint8
	for _, metric := range nonsigners {
		if metric.QuorumId > maxQuorumId {
			maxQuorumId = metric.QuorumId
		}
	}

	result := make([][]string, maxQuorumId+1)

	for _, metric := range nonsigners {
		result[metric.QuorumId] = append(result[metric.QuorumId], metric.OperatorId)
	}
}
