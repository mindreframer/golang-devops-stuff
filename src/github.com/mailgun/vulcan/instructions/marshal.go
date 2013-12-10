/*
This module contains logic unmarshalling rates, upstreams and tokens
from json encoded strings. This logic is pretty verbose, so we are
concentrating it here to keep original modules clean and focusd on the
acutal actions.
*/
package instructions

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type ProxyInstructionsObj struct {
	Failover  Failover
	Tokens    []TokenObj
	Upstreams []UpstreamObj
	Headers   map[string][]string
}

type TokenObj struct {
	Id    string
	Rates []RateObj
}

type RateObj struct {
	Increment int64
	Value     int64
	Period    string
}

// This object is used for unmarshalling from json
type UpstreamObj struct {
	Url     string
	Rates   []RateObj
	Headers map[string][]string
}

func ProxyInstructionsFromJson(bytes []byte) (*ProxyInstructions, error) {
	var r ProxyInstructionsObj
	err := json.Unmarshal(bytes, &r)
	if err != nil {
		return nil, err
	}

	upstreams, err := upstreamsFromJsonList(r.Upstreams)
	if err != nil {
		return nil, err
	}

	tokens, err := tokensFromJsonList(r.Tokens)
	if err != nil {
		return nil, err
	}

	return NewProxyInstructions(&r.Failover, tokens, upstreams, r.Headers)
}

func upstreamsFromJsonList(inUpstreams []UpstreamObj) ([]*Upstream, error) {
	upstreams := make([]*Upstream, len(inUpstreams))
	for i, upstreamObj := range inUpstreams {
		rates, err := ratesFromJsonList(upstreamObj.Rates)
		if err != nil {
			return nil, err
		}
		upstream, err := NewUpstream(
			upstreamObj.Url, rates, upstreamObj.Headers)
		if err != nil {
			return nil, err
		}
		upstreams[i] = upstream
	}
	return upstreams, nil
}

func tokensFromJsonList(inTokens []TokenObj) ([]*Token, error) {
	tokens := make([]*Token, len(inTokens))
	for i, tokenObj := range inTokens {
		rates, err := ratesFromJsonList(tokenObj.Rates)
		if err != nil {
			return nil, err
		}
		token, err := NewToken(tokenObj.Id, rates)
		if err != nil {
			return nil, err
		}
		tokens[i] = token
	}
	return tokens, nil
}

func ratesFromJsonList(inRates []RateObj) ([]*Rate, error) {
	rates := make([]*Rate, len(inRates))
	for i, rateObj := range inRates {
		rate, err := rateFromObj(rateObj)
		if err != nil {
			return nil, err
		}
		rates[i] = rate
	}
	return rates, nil
}

func rateFromObj(obj RateObj) (*Rate, error) {
	period, err := periodFromString(obj.Period)
	if err != nil {
		return nil, err
	}
	return NewRate(obj.Increment, obj.Value, period)
}

//helper to unmarshal periods to golang time.Duration
func periodFromString(period string) (time.Duration, error) {
	switch strings.ToLower(period) {
	case "second":
		return time.Second, nil
	case "minute":
		return time.Minute, nil
	case "hour":
		return time.Hour, nil
	}
	return -1, fmt.Errorf("Unsupported period: %s", period)
}
