package instructions

// Token identifies the request in any
// meaningful way, e.g. Token Id can be account id,
// a combination of account id and ip, etc.
// The idea is that any token can be throttled,
// so we can throttle requests based on account id and ip
// or throttle based on long requests for given service
type Token struct {
	Id    string
	Rates []*Rate
}

func NewToken(id string, rates []*Rate) (*Token, error) {
	return &Token{
		Id:    id,
		Rates: rates,
	}, nil
}

func (t *Token) String() string {
	return t.Id
}
