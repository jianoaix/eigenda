package ejector

import (
	"crypto/ecdsa"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"sort"
	"sync"
	"time"

	"cloud.google.com/go/logging"
	"github.com/Layr-Labs/eigenda/core"
	"github.com/Layr-Labs/eigenda/disperser/dataapi"
)

// The caller should ensure "stakeShare" is in range [0, 1].
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
	if sla >= 1 {
		return 0
	}
	perf := (1 - sla) / nonsigningRate
	return perf / (1.0 + perf)
}

func computePerfScore(metric *OperatorNonsigningPercentageMetrics) {
	return operatorPerfScore(metric.StakePercentage, metric.Percentage/100.0)
}

type Ejector struct {
	Logger     logging.Logger
	Transactor core.Transactor
	privateKey *ecdsa.PrivateKey

	nonsigningRateUri string
	ejectionInterval  time.Duration
	lastEjection      time.Time

	mu sync.Mutex
}

func (e *Ejector) Start() {
	go e.ejectLoop()
}

func (e *Ejector) ejectLoop() {
	ticker := time.NewTicker(e.ejectionInterval)
	defer ticker.Stop()
	for range ticker.C {
		e.eject()
	}
}

func (e *Ejector) handleEjectionRequest(w http.ResponseWriter, r *http.Request) {
	e.eject()
	w.WriteHeader(http.StatusOK)
}

func (e *Ejector) eject() {
	e.mu.Lock()
	defer e.mu.Unlock()

	nonsigners, err := e.getNonsigners()
	if err != nil {
		return
	}

	// Rank the operators for each quorum by the operator performance score.
	// The lower perf score will get ejected with priority in case of rate limiting.
	sort.Slice(nonsigners, func(i, j int) bool {
		if nonsigners[i].QuorumId == nonsigners[j].QuorumId) {
			return computePerfScore(nonsigners[i]) < computePerfScore(nonsigners[j])
		}
		return nonsigners[i].QuorumId < nonsigners[j].QuorumId
	})

	operatorsByQuorum := convertOperators(nonsigners)
}

func (e *Ejection) getNonsigners() ([]*dataapi.OperatorNonsigningPercentageMetrics, error) {
	response, err := http.Get(e.nonsigningRateUri)
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
		if metric.Percentage/100.0 > 1 - stakeShareToSLA(metric.StakePercentage) {
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
