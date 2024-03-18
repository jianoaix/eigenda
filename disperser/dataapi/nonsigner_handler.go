package dataapi

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Layr-Labs/eigenda/core"
	"github.com/Layr-Labs/eigenda/core/eth"
)

func (s *server) getOperatorNonsigningRate(ctx context.Context, intervalSeconds int64) (*OperatorsNonsigningPercentage, error) {
	stageTimer := time.Now()

	batches, err := s.subgraphClient.QueryBatchNonSigningInfoInInterval(ctx, intervalSeconds)
	if err != nil {
		return nil, err
	}
	if len(batches) == 0 {
		return &OperatorsNonsigningPercentage{}, nil
	}

	log := s.logger

	log.Debug("Query subgraph for BatchNonSigningInfo took", "duration:", time.Since(stageTimer).Seconds())
	stageTimer = time.Now()

	// Get the block interval of interest [startBlock, endBlock].
	startBlock := batches[0].ReferenceBlockNumber
	endBlock := batches[0].ReferenceBlockNumber
	for i := range batches {
		if startBlock > batches[i].ReferenceBlockNumber {
			startBlock = batches[i].ReferenceBlockNumber
		}
		if endBlock < batches[i].ReferenceBlockNumber {
			endBlock = batches[i].ReferenceBlockNumber
		}
	}

	// Get the nonsigner (in operatorId) list.
	nonsigners, nonsigningWhen, err := getNonSigners(batches)
	if err != nil {
		return nil, err
	}
	if len(nonsigners) == 0 {
		return &OperatorsNonsigningPercentage{}, nil
	}

	// Get the address for the nonsigners (from their operatorIDs).
	// nonsignerAddresses[i] is the address for nonsigners[i].
	nonsignerAddresses, err := s.transactor.BatchOperatorIDToAddress(ctx, nonsigners)
	if err != nil {
		return nil, err
	}

	nonsignerIdToAddress := make(map[string]string)
	// Create a mapping from address to operatorID.
	nonsignerAddressToId := make(map[string]core.OperatorID)
	for i := range nonsigners {
		nonsignerAddressToId[strings.ToLower(nonsignerAddresses[i].Hex())] = nonsigners[i]
		nonsignerIdToAddress[nonsigners[i].Hex()] = strings.ToLower(nonsignerAddresses[i].Hex())
	}

	log.Debug("Prepare nonsigners took", "duration:", time.Since(stageTimer).Seconds())
	stageTimer = time.Now()

	// Create operators' quorum intervals.
	operatorQuorumIntervals, err := s.createOperatorQuorumIntervals(ctx, nonsigners, nonsigningWhen, nonsignerAddressToId, startBlock, endBlock)
	if err != nil {
		fmt.Println("XXX operatorQuorumIntervals err:", err)
		return nil, err
	}

	log.Debug("Create operator quorum interval took", "duration:", time.Since(stageTimer).Seconds())
	stageTimer = time.Now()

	// Compute num batches failed, where numFailed[op][q] is the number of batches
	// failed to sign for operator "op" and quorum "q".
	numFailed := computeNumFailed(batches, operatorQuorumIntervals)
	log.Debug("Compute num failed", "duration:", time.Since(stageTimer).Seconds())
	stageTimer = time.Now()

	// Compute num batches responsible, where numResponsible[op][q] is the number of batches
	// that operator "op" and quorum "q" are responsible for.
	numResponsible := computeNumResponsible(batches, operatorQuorumIntervals)

	log.Debug("Compute num responsible", "duration:", time.Since(stageTimer).Seconds())
	fmt.Println("XXX numFailed len:", len(numFailed), " num responsible len:", len(numResponsible))
	// stageTimer = time.Now()

	// Compute the nonsigning rate for each <operator, quorum> pair.
	nonsignerMetrics := make([]*OperatorNonsigningPercentageMetrics, 0)
	for op, val := range numResponsible {
		for q, totalCount := range val {
			if totalCount == 0 {
				continue
			}
			if unsignedCount, ok := numFailed[op][q]; ok {
				ps := fmt.Sprintf("%.6f", (float64(unsignedCount)/float64(totalCount))*100)
				pf, err := strconv.ParseFloat(ps, 64)
				if err != nil {
					return nil, err
				}
				address, ok := nonsignerIdToAddress[op]
				if !ok {
					return nil, errors.New("Operator ID and address must be 1:1 mapping")
				}
				nonsignerMetric := OperatorNonsigningPercentageMetrics{
					OperatorId:           fmt.Sprintf("0x%s", op),
					OperatorAddress:      address,
					QuorumId:             q,
					TotalUnsignedBatches: unsignedCount,
					TotalBatches:         totalCount,
					Percentage:           pf,
				}
				nonsignerMetrics = append(nonsignerMetrics, &nonsignerMetric)
			}
		}
	}

	// Sort by descending order of nonsigning rate.
	sort.Slice(nonsignerMetrics, func(i, j int) bool {
		if nonsignerMetrics[i].Percentage == nonsignerMetrics[j].Percentage {
			if nonsignerMetrics[i].OperatorId == nonsignerMetrics[j].OperatorId {
				return nonsignerMetrics[i].QuorumId < nonsignerMetrics[j].QuorumId
			}
			return nonsignerMetrics[i].OperatorId < nonsignerMetrics[j].OperatorId
		}
		return nonsignerMetrics[i].Percentage > nonsignerMetrics[j].Percentage
	})

	fmt.Println("XXX SUCCESS")

	return &OperatorsNonsigningPercentage{
		Meta: Meta{
			Size: len(nonsignerMetrics),
		},
		Data: nonsignerMetrics,
	}, nil
}

func (s *server) createOperatorQuorumIntervals(ctx context.Context, nonsigners []core.OperatorID, nonsigningWhen map[string][]*BatchNonSigningInfo, nonsignerAddressToId map[string]core.OperatorID, startBlock, endBlock uint32) (OperatorQuorumIntervals, error) {
	// Get operators' initial quorums (at startBlock).
	bitmaps, err := s.transactor.GetQuorumBitmapForOperatorsAtBlockNumber(ctx, nonsigners, startBlock)
	if err != nil {
		return nil, err
	}
	operatorInitialQuorum := make(map[string][]uint8)
	for i := range bitmaps {
		operatorInitialQuorum[nonsigners[i].Hex()] = eth.BitmapToQuorumIds(bitmaps[i])
	}
	fmt.Println("XXX operatorInitialQuorum len: ", len(operatorInitialQuorum))

	// Get operators' quorum change events from [startBlock+1, endBlock].
	addedToQuorum, removedFromQuorum, fullSigners, err := s.getOperatorQuorumEvents(ctx, startBlock, endBlock, nonsignerAddressToId)
	if err != nil {
		return nil, err
	}

	fmt.Println("XXXX fullSigners len:", len(fullSigners))
	// Expand beyond nonsigners
	for op, _ := range addedToQuorum {
		if _, ok := operatorInitialQuorum[op]; !ok {
			operatorInitialQuorum[op] = []uint8{}
		}
	}
	for op, _ := range removedFromQuorum {
		if _, ok := operatorInitialQuorum[op]; !ok {
			operatorInitialQuorum[op] = []uint8{}
		}
	}

	// Create operators' quorum intervals.
	operatorQuorumIntervals, err := CreateOperatorQuorumIntervals(startBlock, endBlock, operatorInitialQuorum, addedToQuorum, removedFromQuorum)
	if err != nil {
		return nil, err
	}

	fmt.Println("XXX operatorQuorumIntervals len: ", len(operatorQuorumIntervals))
	for op, val := range operatorQuorumIntervals {
		opHex := fmt.Sprintf("0x%s", op)
		if len(val) == 0 {
			fmt.Println("XXX op:", op, " initial quorums:", operatorInitialQuorum[op], " startBlock:", startBlock, " but has num unsigned batches:", len(nonsigningWhen[opHex]), "An example batch ---- refblock:", nonsigningWhen[opHex][0].ReferenceBlockNumber, " quorum numbers at ref:", nonsigningWhen[opHex][0].QuorumNumbers, " confirmation block:", nonsigningWhen[opHex][0].BlockNumber, "batchId:", nonsigningWhen[opHex][0].BatchId)
			if _, ok := addedToQuorum[op]; ok {
				fmt.Println("XXX op:", op, " quorum added:", addedToQuorum[op])
			} else {
				fmt.Println("XXX op:", op, " quorum added: None")
			}
			if _, ok := removedFromQuorum[op]; ok {
				fmt.Println("XXX op:", op, " quorum removed:", removedFromQuorum[op])
			} else {
				fmt.Println("XXX op:", op, " quorum removed: None")
			}
			// fmt.Println("XXX op:", op, " num quorums:", len(val))
			fmt.Println()
		}
	}

	return operatorQuorumIntervals, nil
}

func (s *server) getOperatorQuorumEvents(ctx context.Context, startBlock, endBlock uint32, nonsignerAddressToId map[string]core.OperatorID) (map[string][]*OperatorQuorum, map[string][]*OperatorQuorum, map[string]struct{}, error) {
	addedToQuorum := make(map[string][]*OperatorQuorum)
	removedFromQuorum := make(map[string][]*OperatorQuorum)
	fullSigners := map[string]struct{}{}
	if startBlock == endBlock {
		return addedToQuorum, removedFromQuorum, fullSigners, nil
	}
	operatorQuorumEvents, err := s.subgraphClient.QueryOperatorQuorumEvent(ctx, startBlock+1, endBlock)
	if err != nil {
		return nil, nil, nil, err
	}
	for op, events := range operatorQuorumEvents.AddedToQuorum {
		if id, ok := nonsignerAddressToId[op]; ok {
			addedToQuorum[id.Hex()] = events
		} else {
			fullSigners[op] = struct{}{}
		}
	}
	for op, events := range operatorQuorumEvents.RemovedFromQuorum {
		if id, ok := nonsignerAddressToId[op]; ok {
			removedFromQuorum[id.Hex()] = events
		} else {
			fullSigners[op] = struct{}{}
		}
	}
	// addedToQuorum = operatorQuorumEvents.AddedToQuorum
	// removedFromQuorum = operatorQuorumEvents.RemovedFromQuorum
	// for k, v := range nonsignerAddressToId {
	// 	fmt.Println("XXX nonsignerAddressToId address:", k, " id:", v.Hex())
	// }
	// Make quorum events organize by operatorID (instead of address) and drop those who
	// are not nonsigners.
	// for op, events := range operatorQuorumEvents.AddedToQuorum {
	// 	if id, ok := nonsignerAddressToId[op]; !ok {
	// 		fullSigners[op] = struct{}{}
	// 		delete(addedToQuorum, op)
	// 	}
	// }
	// for op, events := range operatorQuorumEvents.RemovedFromQuorum {
	// 	if id, ok := nonsignerAddressToId[op]; !ok {
	// 		fullSigners[op] = struct{}{}
	// 		delete(removedFromQuorum, op)
	// 	}
	// }
	fmt.Println("XXX startblock:", startBlock, " endBlock:", endBlock, " len added:", len(addedToQuorum), " len removed:", len(removedFromQuorum))
	// for k, v := range addedToQuorum {
	// 	fmt.Println("XXXX added to quorum, op:", k, " num events:", len(v))
	// }
	// for k, v := range removedFromQuorum {
	// 	fmt.Println("XXXX removed from quorum, op:", k, " num events:", len(v))
	// }

	return addedToQuorum, removedFromQuorum, fullSigners, nil
}

func getNonSigners(batches []*BatchNonSigningInfo) ([]core.OperatorID, map[string][]*BatchNonSigningInfo, error) {
	nonsignerSet := map[string]struct{}{}
	nonsginingWhen := make(map[string][]*BatchNonSigningInfo)
	for _, b := range batches {
		for _, op := range b.NonSigners {
			nonsignerSet[op] = struct{}{}
			bs := nonsginingWhen[op]
			bs = append(bs, b)
			nonsginingWhen[op] = bs
		}
	}
	fmt.Println("XXX nonsginingWhen len:", len(nonsginingWhen))
	nonsigners := make([]core.OperatorID, 0)
	for op := range nonsignerSet {
		hexstr := strings.TrimPrefix(op, "0x")
		b, err := hex.DecodeString(hexstr)
		if err != nil {
			return nil, nil, err
		}
		nonsigners = append(nonsigners, core.OperatorID(b))
	}
	sort.Slice(nonsigners, func(i, j int) bool {
		for k := range nonsigners[i] {
			if nonsigners[i][k] != nonsigners[j][k] {
				return nonsigners[i][k] < nonsigners[j][k]
			}
		}
		return false
	})
	return nonsigners, nonsginingWhen, nil
}

func computeNumFailed(batches []*BatchNonSigningInfo, operatorQuorumIntervals OperatorQuorumIntervals) map[string]map[uint8]int {
	numFailed := make(map[string]map[uint8]int)
	for _, b := range batches {
		for _, op := range b.NonSigners {
			op := op[2:]
			// Note: avg number of quorums per operator is a small number, so use brute
			// force here (otherwise, we can create a map to make it more efficient)
			for _, operatorQuorum := range operatorQuorumIntervals.GetQuorums(op, b.ReferenceBlockNumber) {
				for _, batchQuorum := range b.QuorumNumbers {
					if operatorQuorum == batchQuorum {
						if _, ok := numFailed[op]; !ok {
							numFailed[op] = make(map[uint8]int)
						}
						numFailed[op][operatorQuorum]++
						break
					}
				}
			}
		}
	}
	return numFailed
}

func computeNumResponsible(batches []*BatchNonSigningInfo, operatorQuorumIntervals OperatorQuorumIntervals) map[string]map[uint8]int {
	// Create quorumBatches, where quorumBatches[q].AccuBatches is the total number of
	// batches in block interval [startBlock, b] for quorum "q".
	quorumBatches := CreatQuorumBatches(batches)

	numResponsible := make(map[string]map[uint8]int)
	for op, val := range operatorQuorumIntervals {
		for q, intervals := range val {
			numBatches := 0
			for _, interval := range intervals {
				numBatches = numBatches + ComputeNumBatches(quorumBatches[q], interval.StartBlock, interval.EndBlock)
			}
			if _, ok := numResponsible[op]; !ok {
				numResponsible[op] = make(map[uint8]int)
			}
			numResponsible[op][q] = numBatches
		}
	}
	fmt.Println("XXX operatorQuorumIntervals len:", len(operatorQuorumIntervals), " numResponsible len:", len(numResponsible))
	return numResponsible
}
