package keeper

import (
	"github.com/cosmos/gogoproto/proto"

	"github.com/osmosis-labs/osmosis/osmomath"
	gammtypes "github.com/osmosis-labs/osmosis/v23/x/gamm/types"
	"github.com/osmosis-labs/osmosis/v23/x/superfluid/types"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// This function calculates the melody equivalent worth of an LP share.
// It is intended to eventually use the TWAP of the worth of an LP share
// once that is exposed from the gamm module.
func (k Keeper) calculateMelodyBackingPerShare(pool gammtypes.CFMMPoolI, melodyInPool osmomath.Int) osmomath.Dec {
	twap := melodyInPool.ToLegacyDec().Quo(pool.GetTotalShares().ToLegacyDec())
	return twap
}

func (k Keeper) SetOsmoEquivalentMultiplier(ctx sdk.Context, epoch int64, denom string, multiplier osmomath.Dec) {
	store := ctx.KVStore(k.storeKey)
	prefixStore := prefix.NewStore(store, types.KeyPrefixTokenMultiplier)
	priceRecord := types.OsmoEquivalentMultiplierRecord{
		EpochNumber: epoch,
		Denom:       denom,
		Multiplier:  multiplier,
	}
	bz, err := proto.Marshal(&priceRecord)
	if err != nil {
		panic(err)
	}
	prefixStore.Set([]byte(denom), bz)
}

func (k Keeper) GetSuperfluidMELODYTokens(ctx sdk.Context, denom string, amount osmomath.Int) (osmomath.Int, error) {
	multiplier := k.GetOsmoEquivalentMultiplier(ctx, denom)
	if multiplier.IsZero() {
		return osmomath.ZeroInt(), nil
	}

	decAmt := multiplier.Mul(amount.ToLegacyDec())
	_, err := k.GetSuperfluidAsset(ctx, denom)
	if err != nil {
		return osmomath.ZeroInt(), err
	}
	return k.GetRiskAdjustedMelodyValue(ctx, decAmt.RoundInt()), nil
}

func (k Keeper) DeleteOsmoEquivalentMultiplier(ctx sdk.Context, denom string) {
	store := ctx.KVStore(k.storeKey)
	prefixStore := prefix.NewStore(store, types.KeyPrefixTokenMultiplier)
	prefixStore.Delete([]byte(denom))
}

func (k Keeper) GetOsmoEquivalentMultiplier(ctx sdk.Context, denom string) osmomath.Dec {
	store := ctx.KVStore(k.storeKey)
	prefixStore := prefix.NewStore(store, types.KeyPrefixTokenMultiplier)
	bz := prefixStore.Get([]byte(denom))
	if bz == nil {
		return osmomath.ZeroDec()
	}
	priceRecord := types.OsmoEquivalentMultiplierRecord{}
	err := proto.Unmarshal(bz, &priceRecord)
	if err != nil {
		panic(err)
	}
	return priceRecord.Multiplier
}

func (k Keeper) GetAllOsmoEquivalentMultipliers(ctx sdk.Context) []types.OsmoEquivalentMultiplierRecord {
	store := ctx.KVStore(k.storeKey)
	prefixStore := prefix.NewStore(store, types.KeyPrefixTokenMultiplier)
	iterator := prefixStore.Iterator(nil, nil)
	defer iterator.Close()

	priceRecords := []types.OsmoEquivalentMultiplierRecord{}
	for ; iterator.Valid(); iterator.Next() {
		priceRecord := types.OsmoEquivalentMultiplierRecord{}

		err := proto.Unmarshal(iterator.Value(), &priceRecord)
		if err != nil {
			panic(err)
		}

		priceRecords = append(priceRecords, priceRecord)
	}
	return priceRecords
}
