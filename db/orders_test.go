package db

import (
	"exchange-engine/global"
	"math"
	"testing"
)

func TestCalcChangeAndFees(t *testing.T) {
	// Market Order Situation
	// The price is calculated from existing orders in the orderbook
	ETHUSD := 2417.67
	// The percent difference should be less than 0.01
	tol := 0.01

	bitcloutChange, etherChange, fees := calcChangeAndFees("buy", 10, 150)

	if bitcloutChange != 10*(1-global.Exchange.FEE) {
		t.Fatalf("bitcloutChange is calculated incorrectly. Received: %v. Expected: %v", bitcloutChange, 9.8)
	}
	// Accept a tolerance here because the ETH->USD rate may change slightly between the call above and now
	if math.Abs((etherChange - -(150/ETHUSD))/-(150/ETHUSD)) > tol {
		t.Fatalf("etherChange is calculated incorrectly. Received: %v. Expected: %v", etherChange, -150/ETHUSD)
	}

	if fees != 10*global.Exchange.FEE {
		t.Fatalf("fees are calculated incorrectly. Received: %v. Expected: %v", fees, 10*global.Exchange.FEE)
	}

	bitcloutChange, etherChange, fees = calcChangeAndFees("sell", 10, 150)
	correctFees := 150 * global.Exchange.FEE / ETHUSD
	if bitcloutChange != -10 {
		t.Fatalf("bitcloutChange is calculated incorrectly. Received: %v. Expected: %v", bitcloutChange, -10)
	}
	// Accept a tolerance here because the ETH->USD rate may change slightly between the call above and now
	if math.Abs((etherChange-((150/ETHUSD)-correctFees))/((150/ETHUSD)-correctFees)) > tol {
		t.Fatalf("etherChange is calculated incorrectly. Received: %v. Expected: %v", etherChange, (150/ETHUSD)-correctFees)
	}

	if math.Abs((fees-correctFees)/correctFees) > tol {
		t.Fatalf("fees are calculated incorrectly. Received: %v. Expected: %v", fees, correctFees)
	}
}
