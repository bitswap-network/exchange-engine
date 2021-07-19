package global

import (
	"errors"
	"math/big"
	"strconv"
)

func ToWei(etherValue float64) (weiValue uint64, err error) {
	weiString := strconv.FormatFloat(etherValue*1e18, 'f', 0, 64)
	weiValue, err = strconv.ParseUint(weiString, 10, 64)
	return
}
func FromWei(weiValue float64) (etherValue float64) {
	etherValue = float64(weiValue) / 1e18
	return
}
func ToNanos(cloutValue float64) (nanosValue uint64, err error) {
	nanosString := strconv.FormatFloat(cloutValue*1e9, 'f', 0, 64)
	nanosValue, err = strconv.ParseUint(nanosString, 10, 64)
	return
}
func FromNanos(nanosValue uint64) (cloutValue float64) {
	cloutValue = float64(nanosValue) / 1e9
	return
}

func ToWeiBig(etherValue *big.Float) (weiValue *big.Int, err error) {
	eS := etherValue.Mul(etherValue, big.NewFloat(1e18)).String()
	weiValue = new(big.Int)
	weiValue, ok := weiValue.SetString(eS, 10)
	if !ok {
		return nil, errors.New("set string error")
	}
	return
}
func FromWeiBig(weiValue *big.Int) (etherValue *big.Float, err error) {
	wS := weiValue.String()
	etherValue = new(big.Float)
	etherValue, ok := etherValue.SetString(wS)
	if !ok {
		return nil, errors.New("set string error")
	}
	etherValue = etherValue.Mul(etherValue, big.NewFloat(1e-18))
	return
}

func ToNanosBig(cloutValue *big.Float) (nanosValue *big.Int, err error) {
	cS := cloutValue.Mul(cloutValue, big.NewFloat(1e9)).String()
	nanosValue = new(big.Int)
	nanosValue, ok := nanosValue.SetString(cS, 10)
	if !ok {
		return nil, errors.New("set string error")
	}
	return
}
func FromNanosBig(nanosValue *big.Int) (cloutValue *big.Float, err error) {
	nS := nanosValue.String()
	cloutValue = new(big.Float)
	cloutValue, ok := cloutValue.SetString(nS)
	if !ok {
		return nil, errors.New("set string error")
	}
	cloutValue = cloutValue.Mul(cloutValue, big.NewFloat(1e-9))
	return
}

func ToUSDCBase(usdcValue float64) (usdcBaseValue uint64, err error) {
	usdcBaseString := strconv.FormatFloat(usdcValue*1e6, 'f', 0, 64)
	usdcBaseValue, err = strconv.ParseUint(usdcBaseString, 10, 64)
	return
}
func FromUSDCBase(usdcBaseValue uint64) (usdcValue float64) {
	usdcValue = float64(usdcBaseValue) / 1e6
	return
}
