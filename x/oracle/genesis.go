package oracle

import (
	"fmt"

	"github.com/bandprotocol/bandchain/chain/x/oracle/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	abci "github.com/tendermint/tendermint/abci/types"
)

// GenesisState is the oracle state that must be provided at genesis.
type GenesisState struct {
	Params        types.Params                `json:"params" yaml:"params"` // module level parameters for oracle
	DataSources   []types.DataSource          `json:"data_sources"  yaml:"data_sources"`
	OracleScripts []types.OracleScript        `json:"oracle_scripts"  yaml:"oracle_scripts"`
	ReportInfos   []types.ValidatorReportInfo `json:"report_infos" yaml:"report_infos"`
	Results       [][]byte                    `json:"results" yaml:"results"`
}

// NewGenesisState creates a new genesis state.
func NewGenesisState(
	params types.Params, dataSources []types.DataSource,
	oracleScripts []types.OracleScript, reportInfos []types.ValidatorReportInfo,
	results [][]byte,
) GenesisState {
	return GenesisState{
		Params:        params,
		DataSources:   dataSources,
		OracleScripts: oracleScripts,
		ReportInfos:   reportInfos,
		Results:       results,
	}
}

func ValidateGenesis(data GenesisState) error {
	return nil
}

// DefaultGenesisState returns the default genesis state.
func DefaultGenesisState() GenesisState {
	return GenesisState{
		Params:        types.DefaultParams(),
		DataSources:   []types.DataSource{},
		OracleScripts: []types.OracleScript{},
		ReportInfos:   []types.ValidatorReportInfo{},
		Results:       [][]byte{},
	}
}

func InitGenesis(ctx sdk.Context, k Keeper, data GenesisState) []abci.ValidatorUpdate {
	k.SetParam(ctx, types.KeyMaxRawRequestCount, data.Params.MaxRawRequestCount)
	k.SetParam(ctx, types.KeyGasPerRawDataRequestPerValidator, data.Params.GasPerRawDataRequestPerValidator)
	k.SetParam(ctx, types.KeyExpirationBlockCount, data.Params.ExpirationBlockCount)
	k.SetParam(ctx, types.KeyMaxConsecutiveMisses, data.Params.MaxConsecutiveMisses)
	for _, dataSource := range data.DataSources {
		_ = k.AddDataSource(ctx, types.NewDataSource(
			dataSource.Owner, dataSource.Name, dataSource.Description, dataSource.Filename,
		))
	}

	for _, oracleScript := range data.OracleScripts {
		_ = k.AddOracleScript(ctx, types.NewOracleScript(
			oracleScript.Owner, oracleScript.Name, oracleScript.Description,
			oracleScript.Filename, oracleScript.Schema, oracleScript.SourceCodeURL,
		))
	}

	for _, info := range data.ReportInfos {
		k.SetValidatorReportInfo(ctx, info.Validator, info)
	}

	for i, result := range data.Results {
		if result != nil {
			k.SetResult(ctx, types.RequestID(i+1), result)
		}
	}

	err := k.BindPort(ctx, PortID)
	if err != nil {
		panic(fmt.Sprintf("could not claim port capability: %v", err))
	}
	return []abci.ValidatorUpdate{}
}

func ExportGenesis(ctx sdk.Context, k Keeper) GenesisState {
	return GenesisState{
		Params:        k.GetParams(ctx),
		DataSources:   k.GetAllDataSources(ctx),
		OracleScripts: k.GetAllOracleScripts(ctx),
		ReportInfos:   k.GetAllValidatorReportInfos(ctx),
		Results:       k.GetAllResults(ctx),
	}
}
