package keeper

import (
	"fmt"
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/osmosis-labs/osmosis/v12/x/valset-pref/types"
)

func (k Keeper) SetupValidatorSetPreference(ctx sdk.Context, delegator string, preferences []types.ValidatorPreference) error {
	// check if a user already has a validator-set created
	existingValidators, found := k.GetValidatorSetPreference(ctx, delegator)
	if found {
		// check if the new preferences is the same as the existing preferences
		isEqual := k.IsValidatorSetEqual(preferences, existingValidators.Preferences)
		if isEqual {
			return fmt.Errorf("The preferences (validator and weights) are the same")
		}
	}

	// checks that all the validators exist on chain
	isValid := k.IsPreferenceValid(ctx, preferences)
	if !isValid {
		return fmt.Errorf("The validator preference list is not valid")
	}

	// charge fee to execute this message
	err := k.ChargeForCreateValSet(ctx, delegator)
	if err != nil {
		return err
	}

	return nil
}

// TODO MAYBE: Check if there are any banned assets and ways to handle them
// DelegateToValidatorSet delegates to a delegators existing validator-set.
// For ex: delegate 10osmo with validator-set {ValA -> 0.5, ValB -> 0.3, ValC -> 0.2}
// our delegate logic would attempt to delegate 5osmo to A , 2osmo to B, 3osmo to C
func (k Keeper) DelegateToValidatorSet(ctx sdk.Context, delegatorAddr string, coin sdk.Coin) error {
	// get the existing validator set preference from store
	existingSet, found := k.GetValidatorSetPreference(ctx, delegatorAddr)
	if !found {
		return fmt.Errorf("user %s doesn't have validator set", delegatorAddr)
	}

	delegator, err := sdk.AccAddressFromBech32(delegatorAddr)
	if err != nil {
		return err
	}

	tokenAmt := sdk.NewDec(coin.Amount.Int64())

	// loop through the validatorSetPreference and delegate the proportion of the tokens based on weights
	for _, val := range existingSet.Preferences {
		_, validator, err := k.GetValAddrAndVal(ctx, val.ValOperAddress)
		if err != nil {
			return err
		}

		// amount to delegate, calculated by {val_distribution_weight * tokenAmt}
		// NOTE: it'd be nice if this value was decimal
		amountToStake := val.Weight.Mul(tokenAmt).RoundInt()

		_, err = k.stakingKeeper.Delegate(ctx, delegator, amountToStake, stakingtypes.Unbonded, validator, true)
		if err != nil {
			return err
		}
	}

	return nil
}

// UndelegateFromValidatorSet undelegates {coin} amount from the validator set.
// For ex: userA has staked 10tokens with weight {Val->0.5, ValB->0.3, ValC->0.2}
// undelegate 6osmo with validator-set {ValA -> 0.5, ValB -> 0.3, ValC -> 0.2}
// our undelegate logic would attempt to undelegate 3osmo from A , 1.8osmo from B, 1.2osmo from C
func (k Keeper) UndelegateFromValidatorSet(ctx sdk.Context, delegatorAddr string, coin sdk.Coin) error {
	// get the existing validator set preference
	existingSet, found := k.GetValidatorSetPreference(ctx, delegatorAddr)
	if !found {
		return fmt.Errorf("user %s doesn't have validator set", delegatorAddr)
	}

	delegator, err := sdk.AccAddressFromBech32(delegatorAddr)
	if err != nil {
		return err
	}

	// the total amount the user wants to undelegate
	tokenAmt := sdk.NewDec(coin.Amount.Int64())

	totalAmountFromWeights := sdk.NewDec(0)
	for _, val := range existingSet.Preferences {
		// Calculate the amount to undelegate based on the existing weights
		amountToUnDelegate := val.Weight.Mul(tokenAmt)

		// ValidateValidator gurantees that this exist
		valAddr, err := sdk.ValAddressFromBech32(val.ValOperAddress)
		if err != nil {
			return err
		}

		_, err = k.stakingKeeper.Undelegate(ctx, delegator, valAddr, amountToUnDelegate)
		if err != nil {
			return err
		}

		totalAmountFromWeights = totalAmountFromWeights.Add(amountToUnDelegate)
	}

	if !totalAmountFromWeights.Equal(tokenAmt) {
		return fmt.Errorf("The undelegate total donot add up with the amount calculated from weights expected %s got %s", tokenAmt, totalAmountFromWeights)
	}

	return nil
}

// GetValAddrAndVal checks if the validator address is valid and the validator provided exists on chain.
func (k Keeper) GetValAddrAndVal(ctx sdk.Context, valOperAddress string) (sdk.ValAddress, stakingtypes.Validator, error) {
	valAddr, err := sdk.ValAddressFromBech32(valOperAddress)
	if err != nil {
		return nil, stakingtypes.Validator{}, fmt.Errorf("validator address not formatted")
	}

	validator, found := k.stakingKeeper.GetValidator(ctx, valAddr)
	if !found {
		return nil, stakingtypes.Validator{}, fmt.Errorf("validator not found %s", validator)
	}

	return valAddr, validator, nil
}

// IsPreferenceValid loops through the validator preferences and checks its existence and validity.
func (k Keeper) IsPreferenceValid(ctx sdk.Context, preferences []types.ValidatorPreference) bool {
	for _, val := range preferences {
		_, _, err := k.GetValAddrAndVal(ctx, val.ValOperAddress)
		if err != nil {
			return false
		}
	}
	return true
}

// ChargeForCreateValSet gets the creationFee (default 10osmo) and funds it to the community pool.
func (k Keeper) ChargeForCreateValSet(ctx sdk.Context, delegatorAddr string) error {
	// Send creation fee to community pool
	creationFee := k.GetParams(ctx).ValsetCreationFee
	if creationFee == nil {
		return fmt.Errorf("creation fee cannot be nil or 0 ")
	}

	accAddr, err := sdk.AccAddressFromBech32(delegatorAddr)
	if err != nil {
		return err
	}

	if err := k.distrKeeper.FundCommunityPool(ctx, creationFee, accAddr); err != nil {
		return err
	}

	return nil
}

// IsValidatorSetEqual returns true if the two preferences are equal
func (k Keeper) IsValidatorSetEqual(newPreferences, existingPreferences []types.ValidatorPreference) bool {
	if len(newPreferences) != len(existingPreferences) {
		return false
	}

	sort.Slice(newPreferences, func(i, j int) bool {
		return newPreferences[i].ValOperAddress < newPreferences[j].ValOperAddress
	})

	sort.Slice(existingPreferences, func(i, j int) bool {
		return existingPreferences[i].ValOperAddress < existingPreferences[j].ValOperAddress
	})

	// make sure that both valAddress and weights cannot be the same in the new val-set
	for i := range newPreferences {
		if newPreferences[i].ValOperAddress != existingPreferences[i].ValOperAddress ||
			!newPreferences[i].Weight.Equal(existingPreferences[i].Weight) {
			return false
		}
	}

	return true
}
