package yoda

import (
	"fmt"
	"time"

	sdkCtx "github.com/cosmos/cosmos-sdk/client/context"
	ckeys "github.com/cosmos/cosmos-sdk/client/keys"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/auth/client/utils"

	"github.com/bandprotocol/bandchain/chain/app"
	otypes "github.com/bandprotocol/bandchain/chain/x/oracle/types"
	rpcclient "github.com/tendermint/tendermint/rpc/client"
)

var (
	cdc = app.MakeCodec()
)

func SubmitReport(c *Context, l *Logger, id otypes.RequestID, reps []otypes.RawReport) {
	key := <-c.keys
	defer func() {
		c.keys <- key
	}()

	msg := otypes.NewMsgReportData(otypes.RequestID(id), reps, c.validator, key.GetAddress())
	if err := msg.ValidateBasic(); err != nil {
		l.Error(":exploding_head: Failed to validate basic with error: %s", err.Error())
		return
	}
	cliCtx := sdkCtx.CLIContext{Client: c.client, TrustNode: true, Codec: cdc}
	var res sdk.TxResponse
	success := false
	for try := uint64(1); try <= c.maxTry; try++ {
		acc, err := auth.NewAccountRetriever(cliCtx).GetAccount(key.GetAddress())
		if err != nil {
			l.Debug(":warning: Failed to retreive account with error: %s", err.Error())
			continue
		}

		txBldr := auth.NewTxBuilder(
			auth.DefaultTxEncoder(cdc), acc.GetAccountNumber(), acc.GetSequence(),
			200000, 1, false, cfg.ChainID, "", sdk.NewCoins(), c.gasPrices,
		)
		// txBldr, err = authclient.EnrichWithGas(txBldr, cliCtx, []sdk.Msg{msg})
		// if err != nil {
		// 	l.Error(":exploding_head: Failed to enrich with gas with error: %s", err.Error())
		// 	return
		// }
		out, err := txBldr.WithKeybase(keybase).BuildAndSign(key.GetName(), ckeys.DefaultKeyPass, []sdk.Msg{msg})
		if err != nil {
			l.Debug(":warning: Failed to build tx with error: %s", err.Error())
			continue
		}
		l.Info(":e-mail: Try to broadcast report transaction(%d/%d)", try, c.maxTry)
		res, err = cliCtx.BroadcastTxSync(out)
		if err == nil {
			success = true
			break
		}
		l.Debug(":warning: Failed to broadcast tx with error: %s", err.Error())
		time.Sleep(c.rpcPollIntervall)
	}
	if !success {
		l.Error(":exploding_head: Cannot try to broadcast more than %d try", c.maxTry)
		return
	}
	for start := time.Now(); time.Since(start) < c.broadcastTimeout; {
		time.Sleep(c.rpcPollIntervall)
		txRes, err := utils.QueryTx(cliCtx, res.TxHash)
		if err != nil {
			l.Debug(":warning: Failed to query tx with error: %s", err.Error())
			continue
		}
		if txRes.Code != 0 {
			l.Error(":exploding_head: Tx returned nonzero code %d with log %s, tx hash: %s", txRes.Code, txRes.RawLog, txRes.TxHash)
			return
		}
		l.Info(":smiling_face_with_sunglasses: Successfully broadcast tx with hash: %s", txRes.TxHash)
		return
	}
	l.Info(":question_mark: Cannot get transaction response from hash: %s transaction might be included in the next few blocks or check your node's health.", res.TxHash)
}

// GetExecutable fetches data source executable using the provided client.
func GetExecutable(c *Context, l *Logger, hash string) ([]byte, error) {
	resValue, err := c.fileCache.GetFile(hash)
	if err != nil {
		l.Debug(":magnifying_glass_tilted_left: Fetching data source hash: %s from bandchain querier", hash)
		res, err := c.client.ABCIQueryWithOptions(fmt.Sprintf("custom/%s/%s/%s", otypes.StoreKey, otypes.QueryData, hash), nil, rpcclient.ABCIQueryOptions{})
		if err != nil {
			l.Error(":exploding_head: Failed to get data source with error: %s", err.Error())
			return nil, err
		}
		resValue = res.Response.GetValue()
		c.fileCache.AddFile(resValue)
	} else {
		l.Debug(":card_file_box: Found data source hash: %s in cache file", hash)
	}

	l.Debug(":balloon: Received data source hash: %s content: %q", hash, resValue[:32])
	return resValue, nil
}
