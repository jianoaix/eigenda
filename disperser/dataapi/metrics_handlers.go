package dataapi

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/Layr-Labs/eigenda/core"
)

const (
	avgThroughputWindowSize    = 120 // The time window (in seconds) to calculate the data throughput.
	maxWorkersGetOperatorState = 10  // The maximum number of workers to use when querying operator state.
)

// Representing [startBlock, endBlock] (inclusive).
type BlockInterval struct {
	startBlock, endBlock uint64
}

// OperatorQuorumIntervals[op][q] is a sequence of increasing and non-overlapping
// intervals during which the operator "op" is registered in quorum "q".
type OperatorQuorumIntervals map[string][uint8][]BlockInterval

func removeQuorums(operatorQuorum *OperatorQuorum, openQuorum *map[uint8]uint64, result *OperatorQuorumIntervals) error {
	for _, q := range operatorQuorum.QuorumNumbers {
		start, ok := openQuorum[q]
		if !ok {
			return errors.New("cannot remove a quorum when the operator is not in the quorum")
		}
		if start >= removed[j].BlockNumber {
			return errors.New("deregistration block number must be strictly greater than its registration block number")
		}
		quorumInterval := result[op][quorum]
		interval := BlockInterval{
			startBlock: start,
			// The operator is NOT live at the block it's deregistered.
			endBlock: removed[j].BlockNumber - 1,
		}
		quorumInterval = append(quorumInterval, interval)
		result[op][quorum] = quorumInterval
		delete(openQuorum, q)
	}
}

func createOperatorQuorumIntervals(
	queryStartBlock uint64,
	queryEndBlock uint64,
	operatorQuorum map[string][]uint8,
	addedToQuorum map[string][]*OperatorQuorum,
	removedFromQuorum map[string][]*OperatorQuorum,
) (*OperatorQuorumIntervals, error) {
	operatorQuorumInterval = make(OperatorQuorumIntervals)

	// operatorQuorum is the initial quorums for the nonsigning operators.
	for op, quorums := range operatorQuorum {
		openQuorum := make(map[uint8]uint64)
		for _, q := range quorums {
			openQuorum[q] = queryStartBlock
		}
		removed := removedFromQuorum[op]
		added := addedToQuorum[op]
		i, j : = 0, 0
		for i < len(added) && j < len(removed) {
			if added[i].BlockNumber < removed[j].BlockNumber {
				for _, q := range added[i].QuorumNumbers {
					start, ok := openQuorums[q]
					if ok {
						return nil, errors.New("cannot add to quorum when the operator is already in the quorum")
					}
					openQuorum[q] = start
				}
				i++
			} else {
				err := removeQuorums(removed[j], openQuorum, operatorQuorumInterval)
				if err != nil {
					return nil, err
				}
				j++
			}
		}
		for ; i < len(addded); i++ {
			for _, q := range added[i].QuorumNumbers {
				start, ok := openQuorums[q]
				if ok {
					return nil, errors.New("cannot add to quorum when the operator is already in the quorum")
				}
				openQuorum[q] = start
			}
			i++
		}
		for ; j < len(removed); j++ {
			err := removeQuorums(removed[j], openQuorum, operatorQuorumInterval)
			if err != nil {
				return nil, err
			}
		}
		for q, start := range openQuorum {
			interval := BlockInterval{
				startBlock: start,
				endBlock: queryEndBlock,
			}
			operatorQuorumInterval[op][q] = append(operatorQuorumInterval[op][q], interval)
		}
	}

	return operatorQuorumInterval, nil
}

func getQuorums(operatorId string, blockNum uint64) []uint8 {
	quorums := make([]uint8)
	for op, quorumIntervals := range opi.operatorQuorumIntervals {
		for q, intervals := range quorumIntervals {
			// TODO: if len(intervals) is large, we can perform binary search here.
			live := false
			for _, interval := range intervals {
				if interval.startBlock > blockNum {
					break;
				}
				if blockNum <=interval.endBlock {
					live = true
					break
				}
			}
			if live {
				quorums = append(quorums, q)
			}
		}
	}
	return quorums
}

func (s *server) fetchOperatorAddress(ctx context.Context, nonSigners []*BatchNonSigningInfo) error {
	pool = workerpool.New(maxWorkerPoolSize)
	err Error
	for _, batch := range batches {
		pool.Submit(func() {
			address, errQ := s.transactor.OperatorIDToAddress(ctx, batch.OperatorId)
			if errQ != nil {
				err = errQ
			}
		})
	}
	pool.StopWait()

	return err
}

func (s *server) getMetric(ctx context.Context, startTime int64, endTime int64, limit int) (*Metric, error) {
	blockNumber, err := s.transactor.GetCurrentBlockNumber(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current block number: %w", err)
	}
	operatorState, err := s.chainState.GetOperatorState(ctx, uint(blockNumber), []core.QuorumID{core.QuorumID(0)})
	if err != nil {
		return nil, err
	}
	if len(operatorState.Operators) != 1 {
		return nil, fmt.Errorf("Requesting for one quorum (quorumID=0), but got %v", operatorState.Operators)
	}
	totalStake := big.NewInt(0)
	for _, op := range operatorState.Operators[0] {
		totalStake.Add(totalStake, op.Stake)
	}

	result, err := s.promClient.QueryDisperserBlobSizeBytesPerSecond(ctx, time.Unix(startTime, 0), time.Unix(endTime, 0))
	if err != nil {
		return nil, err
	}

	var (
		totalBytes   float64
		timeDuration float64
		troughput    float64
		valuesSize   = len(result.Values)
	)
	if valuesSize > 1 {
		totalBytes = result.Values[valuesSize-1].Value - result.Values[0].Value
		timeDuration = result.Values[valuesSize-1].Timestamp.Sub(result.Values[0].Timestamp).Seconds()
		troughput = totalBytes / timeDuration
	}

	costInGas, err := s.calculateTotalCostGasUsed(ctx)
	if err != nil {
		return nil, err
	}

	return &Metric{
		Throughput: troughput,
		CostInGas:  costInGas,
		TotalStake: totalStake.Uint64(),
	}, nil
}

func (s *server) getThroughput(ctx context.Context, start int64, end int64) ([]*Throughput, error) {
	result, err := s.promClient.QueryDisperserAvgThroughputBlobSizeBytes(ctx, time.Unix(start, 0), time.Unix(end, 0), avgThroughputWindowSize)
	if err != nil {
		return nil, err
	}

	if len(result.Values) <= 1 {
		return []*Throughput{}, nil
	}

	throughputs := make([]*Throughput, 0)
	for i := avgThroughputWindowSize; i < len(result.Values); i++ {
		v := result.Values[i]
		throughputs = append(throughputs, &Throughput{
			Timestamp:  uint64(v.Timestamp.Unix()),
			Throughput: v.Value,
		})
	}

	return throughputs, nil
}

func (s *server) calculateTotalCostGasUsed(ctx context.Context) (float64, error) {
	batches, err := s.subgraphClient.QueryBatchesWithLimit(ctx, 1, 0)
	if err != nil {
		return 0, err
	}

	if len(batches) == 0 {
		return 0, nil
	}

	var (
		totalBlobSize uint
		totalGasUsed  float64
		batch         = batches[0]
	)

	if batch == nil {
		return 0, errors.New("error the latest batch is not valid")
	}

	batchHeaderHash, err := ConvertHexadecimalToBytes(batch.BatchHeaderHash)
	if err != nil {
		s.logger.Error("Failed to convert BatchHeaderHash to hex string: ", "batchHeaderHash", batch.BatchHeaderHash, "err", err)
		return 0, err
	}

	metadatas, err := s.blobstore.GetAllBlobMetadataByBatch(ctx, batchHeaderHash)
	if err != nil {
		s.logger.Error("Failed to get all blob metadata by batch: ", "batchHeaderHash", batchHeaderHash, "err", err)
		return 0, err
	}

	for _, metadata := range metadatas {
		totalBlobSize += metadata.RequestMetadata.BlobSize
	}

	if uint64(totalBlobSize) > 0 {
		totalGasUsed = float64(batch.GasFees.GasUsed) / float64(totalBlobSize)
	}
	return totalGasUsed, nil
}

func (s *server) getNonSigners(ctx context.Context, intervalSeconds int64) (*[]NonSigner, error) {
	nonSigners, err := s.subgraphClient.QueryBatchNonSigningOperatorIdsInInterval(ctx, intervalSeconds)
	if err != nil {
		return nil, err
	}

	nonSignersObj := make([]NonSigner, 0)
	for nonSigner, nonSigningAmount := range nonSigners {
		s.logger.Info("NonSigner", "nonSigner", nonSigner, "nonSigningAmount", nonSigningAmount)
		nonSignersObj = append(nonSignersObj, NonSigner{
			OperatorId: nonSigner,
			Count:      nonSigningAmount,
		})
	}

	return &nonSignersObj, nil
}

func (s *server) getOperatorNonsigningPercentage(ctx context.Context, intervalSeconds int64) (*OperatorsNonsigningPercentage, error) {
	batches, err := s.subgraphClient.QueryBatchNonSigningInfoInInterval(ctx, intervalSeconds)
	if err != nil {
		return nil, err
	}

	if len(batches) == 0 {
		return &OperatorsNonsigningPercentage{}, nil
	}

	err := fetchOperatorAddress(batches)
	if err != nil {
		return nil, err
	}

	startBlock := batches[0].ReferenceBlockNumber
	endBlock := batches[0].ReferenceBlockNumber
	for i := range batches {
		if startBlock > blockNum {
			startBlock = blockNum
		}
		if endBlock < blockNum {
			endBlock = blockNum
		}
	}

	return &OperatorsNonsigningPercentage{
	}, nil
}

func createOperatorQuorumInfo(ctx, nonSigners []string, startBlock, endBlock uint64) (*OperatorQuorumInfo, error) {
	quorumEvents, err := s.subgraphClient.QueryOperatorQuorumEvent(ctx, startBlock, endBlock)
	if err != nil {
		return nil, err
	}
}

// func (s *server) getOperatorNonsigningPercentage(ctx context.Context, intervalSeconds int64) (*OperatorsNonsigningPercentage, error) {
// 	nonSigners, err := s.subgraphClient.QueryBatchNonSigningOperatorIdsInInterval(ctx, intervalSeconds)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	if len(nonSigners) == 0 {
// 		return &OperatorsNonsigningPercentage{}, nil
// 	}
//
// 	pastBlockTimestamp := uint64(time.Now().Add(-time.Duration(intervalSeconds) * time.Second).Unix())
// 	numBatchesByOperators, err := s.subgraphClient.QueryNumBatchesByOperatorsInThePastBlockTimestamp(ctx, pastBlockTimestamp, nonSigners)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	if len(numBatchesByOperators) == 0 {
// 		return &OperatorsNonsigningPercentage{}, nil
// 	}
//
// 	operators := make([]*OperatorNonsigningPercentageMetrics, 0)
// 	for operatorId, totalUnsignedBatches := range nonSigners {
// 		if totalUnsignedBatches > 0 {
// 			numBatches := numBatchesByOperators[operatorId]
// 			if numBatches == 0 {
// 				continue
// 			}
// 			ps := fmt.Sprintf("%.2f", (float64(totalUnsignedBatches)/float64(numBatches))*100)
// 			pf, err := strconv.ParseFloat(ps, 64)
// 			if err != nil {
// 				return nil, err
// 			}
// 			operatorMetric := OperatorNonsigningPercentageMetrics{
// 				OperatorId:           operatorId,
// 				TotalUnsignedBatches: totalUnsignedBatches,
// 				TotalBatches:         numBatches,
// 				Percentage:           pf,
// 			}
// 			operators = append(operators, &operatorMetric)
// 		}
// 	}
//
// 	// Sort by descending order of nonsigning rate.
// 	sort.Slice(operators, func(i, j int) bool {
// 		if operators[i].Percentage == operators[j].Percentage {
// 			return operators[i].OperatorId < operators[j].OperatorId
// 		}
// 		return operators[i].Percentage > operators[j].Percentage
// 	})
//
// 	return &OperatorsNonsigningPercentage{
// 		Meta: Meta{
// 			Size: len(operators),
// 		},
// 		Data: operators,
// 	}, nil
// }
