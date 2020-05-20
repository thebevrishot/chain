package db

import (
	"strconv"

	"github.com/bandprotocol/bandchain/chain/x/oracle"
	otypes "github.com/bandprotocol/bandchain/chain/x/oracle/types"
)

const (
	Open    = "Pending"
	Success = "Success"
	Failure = "Failure"
	Expired = "Expired"
	Unknown = "Unknown"
)

func parseResolveStatus(resolveStatus otypes.ResolveStatus) string {
	switch resolveStatus {
	case 0:
		return Open
	case 1:
		return Success
	case 2:
		return Failure
	case 3:
		return Expired
	default:
		return Unknown
	}
}

func createRequest(
	id int64,
	oracleScriptID int64,
	calldata []byte,
	minCount int64,
	expirationHeight int64,
	resolveStatus string,
	requester string,
	clientID string,
	txHash []byte,
	result []byte,
) Request {
	return Request{
		ID:               id,
		OracleScriptID:   oracleScriptID,
		Calldata:         calldata,
		MinCount:         minCount,
		ExpirationHeight: expirationHeight,
		ResolveStatus:    resolveStatus,
		Requester:        requester,
		ClientID:         clientID,
		TxHash:           txHash,
		Result:           result,
	}
}

func (b *BandDB) AddNewRequest(
	id int64,
	oracleScriptID int64,
	calldata []byte,
	minCount int64,
	expirationHeight int64,
	resolveStatus string,
	requester string,
	clientID string,
	txHash []byte,
	result []byte,
) error {
	request := createRequest(
		id,
		oracleScriptID,
		calldata,
		minCount,
		expirationHeight,
		resolveStatus,
		requester,
		clientID,
		txHash,
		result,
	)
	err := b.tx.Create(&request).Error
	if err != nil {
		return err
	}

	req, err := b.OracleKeeper.GetRequest(b.ctx, otypes.RequestID(id))
	if err != nil {
		return err
	}

	for _, validatorAddress := range req.RequestedValidators {
		err := b.AddRequestedValidator(id, validatorAddress.String())
		if err != nil {
			return err
		}
	}

	// TODO: FIX ME. Bun please take care of this
	// for _, raw := range b.OracleKeeper.GetRawRequests(b.ctx, otypes.RequestID(id)) {
	// 	err := b.AddRawDataRequest(
	// 		id,
	// 		int64(raw.ExternalID),
	// 		int64(raw.DataSourceID),
	// 		raw.Calldata,
	// 	)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	err = b.tx.FirstOrCreate(&RelatedDataSources{
	// 		DataSourceID:   int64(raw.DataSourceID),
	// 		OracleScriptID: int64(oracleScriptID),
	// 	}).Error
	// 	if err != nil {
	// 		return err
	// 	}
	// }

	return nil
}

func createRequestedValidator(
	requestID int64,
	validatorAddress string,
) RequestedValidator {
	return RequestedValidator{
		RequestID:        requestID,
		ValidatorAddress: validatorAddress,
	}
}

func (b *BandDB) AddRequestedValidator(
	requestID int64,
	validatorAddress string,
) error {
	requestValidator := createRequestedValidator(
		requestID,
		validatorAddress,
	)
	err := b.tx.Create(&requestValidator).Error
	return err
}

func createRawDataRequests(
	requestID int64,
	externalID int64,
	dataSourceID int64,
	calldata []byte,
) RawDataRequests {
	return RawDataRequests{
		RequestID:    requestID,
		ExternalID:   externalID,
		DataSourceID: dataSourceID,
		Calldata:     calldata,
	}
}

func (b *BandDB) AddRawDataRequest(
	requestID int64,
	externalID int64,
	dataSourceID int64,
	calldata []byte,
) error {
	rawDataRequests := createRawDataRequests(
		requestID,
		externalID,
		dataSourceID,
		calldata,
	)
	err := b.tx.Create(&rawDataRequests).Error
	return err
}

func (b *BandDB) handleMsgRequestData(
	txHash []byte,
	msg oracle.MsgRequestData,
	events map[string]string,
) error {
	id, err := strconv.ParseInt(events[otypes.EventTypeRequest+"."+otypes.AttributeKeyID], 10, 64)
	if err != nil {
		return err
	}
	request, err := b.OracleKeeper.GetRequest(b.ctx, otypes.RequestID(id))
	if err != nil {
		return err
	}
	return b.AddNewRequest(
		id,
		int64(msg.OracleScriptID),
		msg.Calldata,
		msg.MinCount,
		request.RequestHeight+20, // TODO: REMOVE THIS. HACK!!!
		"Pending",
		msg.Sender.String(),
		msg.ClientID,
		txHash,
		nil,
	)
}
