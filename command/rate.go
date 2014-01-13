package command

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

var rateRe *regexp.Regexp

func init() {
	rateRe = regexp.MustCompile(`(?P<requests>\d+) (?P<unit>(?:req|reqs|request|requests|KB))?/(?P<period>(?:second|minute|hour))`)
}

const (
	// Limit type by the amount of requests in a given time period.
	UnitTypeRequests = iota
	// Limit type by the amount by used bandwidth.
	UnitTypeKilobytes = iota
)

// Rates  the information on how many hits or bytes
// per period of time any endpoint can accept.
type Rate struct {
	// The amount of unit (e.g. 100 for 100 Kilobytes)
	Units int64
	// Rate's period of time (e.g. second in 100 request/second)
	Period time.Duration
	// Unit to measure, e.g. requests or kilobytes
	UnitType int
}

func NewRate(units int64, period time.Duration, unitType int) (*Rate, error) {
	if units <= 0 {
		return nil, fmt.Errorf("unit value should be > 0")
	}
	if unitType != UnitTypeRequests && unitType != UnitTypeKilobytes {
		return nil, fmt.Errorf("Unsupported unit type: %v", unitType)
	}
	if period < time.Second || period > 24*time.Hour {
		return nil, fmt.Errorf("Period should be within [1 second, 24 hours]")
	}
	return &Rate{Units: units, Period: period, UnitType: unitType}, nil
}

func (r *Rate) String() string {
	return fmt.Sprintf("Rate(units=%d, unitType=%s, period=%s)", r.Units, UnitTypeToString(r.UnitType), r.Period)
}

// Calculates when this rate can be hit the next time from
// the given time t, assuming all the requests in the given
func (r *Rate) RetrySeconds(now time.Time) int {
	return int(r.NextBucket(now).Unix() - now.Unix())
}

//Returns epochSeconds rounded to the rate period
//e.g. minutes rate would return epoch seconds with seconds set to zero
//hourly rate would return epoch seconds with minutes and seconds set to zero
func (r *Rate) CurrentBucket(t time.Time) time.Time {
	return t.Truncate(r.Period)
}

// Returns the epoch seconds of the begining of the next time bucket
func (r *Rate) NextBucket(t time.Time) time.Time {
	return r.CurrentBucket(t.Add(r.Period))
}

func NewRatesFromObj(in interface{}) (map[string][]*Rate, error) {
	rates := make(map[string][]*Rate)
	ratesM, ok := in.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("Expected dictionary with rates, got %T", in)
	}

	for key, ratesI := range ratesM {
		switch ratesC := ratesI.(type) {
		case []interface{}:
			vals := make([]*Rate, len(ratesC))
			for i, rateI := range ratesC {
				rate, err := NewRateFromObj(rateI)
				if err != nil {
					return nil, err
				}
				vals[i] = rate
			}
			rates[key] = vals
		case interface{}:
			vals := make([]*Rate, 1)
			rate, err := NewRateFromObj(ratesC)
			if err != nil {
				return nil, err
			}
			vals[0] = rate
			rates[key] = vals
		default:
			return nil, fmt.Errorf("Unsupported type in rate definition: %T", ratesC)
		}
	}
	return rates, nil
}

func NewRateFromObj(in interface{}) (*Rate, error) {
	switch val := in.(type) {
	case string:
		return NewRateFromString(val)
	case map[string]interface{}:
		return NewRateFromDict(val)
	default:
		return nil, fmt.Errorf("Rate string or dict required")
	}
}

func NewRateFromString(in string) (*Rate, error) {
	values := rateRe.FindStringSubmatch(in)
	if values == nil {
		return nil, fmt.Errorf("Unsupported rate format")
	}
	requests, err := strconv.Atoi(values[1])
	if err != nil {
		return nil, fmt.Errorf("Rate requests should be integer")
	}
	unit, err := UnitTypeFromString(values[2])
	if err != nil {
		return nil, err
	}
	period, err := PeriodFromString(values[3])
	if err != nil {
		return nil, err
	}
	return NewRate(int64(requests), period, unit)
}

func NewRateFromDict(in map[string]interface{}) (*Rate, error) {
	units, unitType, err := getUnitAndValue(in)
	if err != nil {
		return nil, err
	}
	periodI, ok := in["period"]
	if !ok {
		return nil, fmt.Errorf("Expected period")
	}
	periodS, ok := periodI.(string)
	if !ok {
		return nil, fmt.Errorf("Period should be a string")
	}
	period, err := PeriodFromString(periodS)
	if err != nil {
		return nil, err
	}
	return NewRate(int64(units), period, unitType)
}

func getUnitAndValue(in map[string]interface{}) (int, int, error) {
	requestsI, ok := in["requests"]
	if ok {
		units, err := getInt("requests", requestsI)
		return units, UnitTypeRequests, err
	}
	kilobytesI, ok := in["KB"]
	if ok {
		units, err := getInt("KB", kilobytesI)
		return units, UnitTypeKilobytes, err
	}
	return -1, -1, fmt.Errorf("Unsupported unit")
}

func getInt(name string, in interface{}) (int, error) {
	inF, ok := in.(float64)
	if !ok || inF != float64(int(inF)) {
		return -1, fmt.Errorf("Parameter '%s' should be integer", name)
	}
	return int(inF), nil
}

func UnitTypeFromString(u string) (int, error) {
	switch u {
	case "KB":
		return UnitTypeKilobytes, nil
	case "req", "reqs", "requests", "request":
		return UnitTypeRequests, nil
	default:
		return -1, fmt.Errorf("Unsupported unit")
	}
}

func UnitTypeToString(u int) string {
	switch u {
	case UnitTypeRequests:
		return "requests"
	case UnitTypeKilobytes:
		return "KB"
	default:
		return "<error:unsupported unit type>"
	}
}

func PeriodFromString(u string) (time.Duration, error) {
	switch u {
	case "second":
		return time.Second, nil
	case "minute":
		return time.Minute, nil
	case "hour":
		return time.Hour, nil
	default:
		return -1, fmt.Errorf("Unsupported period: %s", u)
	}
}
