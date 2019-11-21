/*

 */

package event

import (
	"crypto/rand"
	"io"

	"github.com/Oneledger/protocol/action"
	"github.com/Oneledger/protocol/action/btc"
	"github.com/Oneledger/protocol/chains/bitcoin"
	"github.com/Oneledger/protocol/data/jobs"
)

type JobBTCCheckFinality struct {
	Type string

	TrackerName string
	JobID       string

	Status jobs.Status
}

func NewBTCCheckFinalityJob(trackerName, id string) jobs.Job {

	return &JobBTCCheckFinality{
		Type:        JobTypeBTCCheckFinality,
		TrackerName: trackerName,
		JobID:       id,
		Status:      jobs.New,
	}
}

func (cf *JobBTCCheckFinality) DoMyJob(ctxI interface{}) {

	ctx, _ := ctxI.(*JobsContext)

	tracker, err := ctx.Trackers.Get(cf.TrackerName)
	if err != nil {
		ctx.Logger.Error("err trying to deserialize tracker: ", cf.TrackerName, err)
		return
	}

	cd := bitcoin.NewChainDriver(ctx.BlockCypherToken)

	chain := "test3"
	switch ctx.BTCChainnet {
	case "testnet3":
		chain = "test3"
	case "testnet":
		chain = "test"
	case "mainnet":
		chain = "main"
	}

	// tempHash, _ := chainhash.NewHashFromStr("860a32ef84ed54df86d207112d1f8d3d5ad28751b25cc7e2107ef55cccbc7586")
	// ok, err := cd.CheckFinality(tempHash, ctx.BlockCypherToken, chain)

	ok, err := cd.CheckFinality(tracker.ProcessTxId, ctx.BlockCypherToken, chain)
	if err != nil {
		ctx.Logger.Error("error while checking finality", err, cf.TrackerName)
		return
	}

	if !ok {
		ctx.Logger.Info("not finalized yet", cf.TrackerName)
		return
	}

	data := [4]byte{}
	_, err = io.ReadFull(rand.Reader, data[:])
	if err != nil {
		ctx.Logger.Error("error while reading random bytes for minting", err, cf.TrackerName)
		return
	}

	reportFinalityMint := btc.ReportFinalityMint{
		TrackerName:      cf.TrackerName,
		OwnerAddress:     tracker.ProcessOwner,
		ValidatorAddress: ctx.ValidatorAddress,
		RandomBytes:      data[:],
	}

	txData, err := reportFinalityMint.Marshal()
	if err != nil {
		ctx.Logger.Error("error while preparing mint txn ", err, cf.TrackerName)
		return
	}

	tx := action.RawTx{
		Type: action.BTC_REPORT_FINALITY_MINT,
		Data: txData,
		Fee:  action.Fee{},
		Memo: cf.JobID,
	}

	req := InternalBroadcastRequest{
		RawTx: tx,
	}
	rep := BroadcastReply{}

	err = ctx.Service.InternalBroadcast(req, &rep)
	if err != nil {
		ctx.Logger.Error("error while broadcasting finality vote and mint txn ", err, cf.TrackerName)
		return
	}

	cf.Status = jobs.Completed
}

func (cf *JobBTCCheckFinality) GetType() string {
	return JobTypeBTCCheckFinality
}

func (cf *JobBTCCheckFinality) GetJobID() string {
	return cf.JobID
}

func (cf *JobBTCCheckFinality) IsDone() bool {
	return cf.Status == jobs.Completed
}
