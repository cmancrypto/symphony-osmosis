package treasury

import (
	appparams "github.com/osmosis-labs/osmosis/v23/app/params"
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/osmosis-labs/osmosis/v23/x/treasury/keeper"
)

func TestExportInitGenesis(t *testing.T) {
	input := keeper.CreateTestInput(t)
	input.Ctx = input.Ctx.WithBlockHeight(int64(appparams.BlocksPerWeek) * 3)

	input.TreasuryKeeper.SetTaxRate(input.Ctx, sdk.NewDec(5435))
	genesis := ExportGenesis(input.Ctx, input.TreasuryKeeper)

	newInput := keeper.CreateTestInput(t)
	newInput.Ctx = newInput.Ctx.WithBlockHeight(int64(appparams.BlocksPerWeek) * 3)
	InitGenesis(newInput.Ctx, newInput.TreasuryKeeper, genesis)
	newGenesis := ExportGenesis(newInput.Ctx, newInput.TreasuryKeeper)

	require.Equal(t, genesis, newGenesis)

	newInput = keeper.CreateTestInput(t)
	newInput.Ctx = newInput.Ctx.WithBlockHeight(int64(appparams.BlocksPerWeek) * 3)
	InitGenesis(newInput.Ctx, newInput.TreasuryKeeper, genesis)
	newGenesis = ExportGenesis(newInput.Ctx, newInput.TreasuryKeeper)
}
