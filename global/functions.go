package global

import "strconv"

func ToWei(etherValue float64) (weiValue uint64, err error) {
	weiString := strconv.FormatFloat(etherValue*1e18, 'E', -1, 64)
	weiValue, err = strconv.ParseUint(weiString, 10, 64)
	return
}
func FromWei(weiValue uint64) (etherValue float64) {
	etherValue = float64(weiValue) / 1e18
	return
}
func ToNanos(cloutValue float64) (nanosValue uint64, err error) {
	nanosString := strconv.FormatFloat(cloutValue*1e9, 'E', -1, 64)
	nanosValue, err = strconv.ParseUint(nanosString, 10, 64)
	return
}
func FromNanos(nanosValue uint64) (cloutValue float64) {
	cloutValue = float64(nanosValue) / 1e9
	return
}
func ToUSDCBase(usdcValue float64) (usdcBaseValue uint64, err error) {
	usdcBaseString := strconv.FormatFloat(usdcValue*1e6, 'E', -1, 64)
	usdcBaseValue, err = strconv.ParseUint(usdcBaseString, 10, 64)
	return
}
func FromUSDCBase(usdcBaseValue uint64) (usdcValue float64) {
	usdcValue = float64(usdcBaseValue) / 1e6
	return
}
