package keeper

import (
	"github.com/bandprotocol/bandchain/chain/x/oracle/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// HasReport checks if the report of this ID triple exists in the storage.
func (k Keeper) HasReport(ctx sdk.Context, rid types.RequestID, val sdk.ValAddress) bool {
	return ctx.KVStore(k.storeKey).Has(types.RawDataReportStoreKey(rid, val))
}

// SetDataReport saves the report to the storage without performing validation.
func (k Keeper) SetReport(ctx sdk.Context, rid types.RequestID, rep types.Report) {
	key := types.RawDataReportStoreKey(rid, rep.Validator)
	ctx.KVStore(k.storeKey).Set(key, k.cdc.MustMarshalBinaryBare(rep))
}

// AddReports performs sanity checks and adds a new batch from one validator to one request
// to the store. Note that we expect each validator to report to all raw data requests at once.
func (k Keeper) AddReport(ctx sdk.Context, rid types.RequestID, rep types.Report) error {
	req, err := k.GetRequest(ctx, rid)
	if err != nil {
		return err
	}
	if !ContainsVal(req.RequestedValidators, rep.Validator) {
		return sdkerrors.Wrapf(
			types.ErrValidatorNotRequested, "reqID: %d, val: %s", rid, rep.Validator.String())
	}
	if k.HasReport(ctx, rid, rep.Validator) {
		return sdkerrors.Wrapf(
			types.ErrValidatorAlreadyReported, "reqID: %d, val: %s", rid, rep.Validator.String())
	}
	if len(rep.RawReports) != len(req.RawRequestIDs) {
		return types.ErrInvalidDataSourceCount
	}
	for _, rep := range rep.RawReports {
		// Here we can safely assume that external IDs are unique, as this has already been
		// checked by ValidateBasic performed in baseapp's runTx function.
		if !ContainsEID(req.RawRequestIDs, rep.ExternalID) {
			return sdkerrors.Wrapf(
				types.ErrRawRequestNotFound, "reqID: %d, extID: %d", rid, rep.ExternalID)
		}
	}
	k.SetReport(ctx, rid, rep)
	return nil
}

// GetReportIterator returns the iterator for all reports of the given request ID.
func (k Keeper) GetReportIterator(ctx sdk.Context, rid types.RequestID) sdk.Iterator {
	prefix := types.GetIteratorPrefix(types.RawDataReportStoreKeyPrefix, rid)
	return sdk.KVStorePrefixIterator(ctx.KVStore(k.storeKey), prefix)
}

// GetReportCount returns the number of reports for the given request ID.
func (k Keeper) GetReportCount(ctx sdk.Context, rid types.RequestID) (count int64) {
	iterator := k.GetReportIterator(ctx, rid)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		count++
	}
	return count
}

// GetReports returns all reports for the given request ID, or nil if there is none.
func (k Keeper) GetReports(ctx sdk.Context, rid types.RequestID) (reports []types.Report) {
	iterator := k.GetReportIterator(ctx, rid)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		var rep types.Report
		k.cdc.MustUnmarshalBinaryBare(iterator.Value(), &rep)
		reports = append(reports, rep)
	}
	return reports
}

// DeleteReports removes all reports for the given request ID.
func (k Keeper) DeleteReports(ctx sdk.Context, rid types.RequestID) {
	var keys [][]byte
	iterator := k.GetReportIterator(ctx, rid)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		keys = append(keys, iterator.Key())
	}
	for _, key := range keys {
		ctx.KVStore(k.storeKey).Delete(key)
	}
}

// UpdateReportInfos updates validator report info for jail validator
// that miss report more than threshold.
func (k Keeper) UpdateReportInfos(ctx sdk.Context, rid types.RequestID) {
	reportedMap := make(map[string]bool)
	reports := k.GetReports(ctx, rid)
	for _, report := range reports {
		reportedMap[report.Validator.String()] = true
	}
	request := k.MustGetRequest(ctx, rid)
	for _, val := range request.RequestedValidators {
		_, voted := reportedMap[val.String()]
		k.HandleValidatorReport(ctx, val, voted)
	}
}
