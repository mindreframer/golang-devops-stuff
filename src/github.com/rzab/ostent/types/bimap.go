package types

type Seq2string map[SEQ]string
type Biseqmap struct {
	SEQ2STRING Seq2string
	STRING2SEQ map[string]SEQ
	SEQ2REVERSE map[SEQ]bool
	Default_seq SEQ
}

func contains(thiss SEQ, lists []SEQ) bool {
	for _, s := range lists {
		if s == thiss {
			return true
		}
	}
	return false
}

func Seq2bimap(def_seq SEQ, s2s Seq2string, reverse []SEQ) Biseqmap {
	bi := Biseqmap{
		SEQ2STRING:  Seq2string    {},
		STRING2SEQ:  map[string]SEQ{},
		SEQ2REVERSE: map[SEQ]bool  {},
	}
	bi.Default_seq = def_seq

	for seq, str := range s2s {
		isreverse := contains(seq, reverse)
		bi.SEQ2REVERSE[ seq] = isreverse
		bi.SEQ2REVERSE[-seq] = isreverse

		bi.SEQ2STRING[ seq] =      str
		bi.SEQ2STRING[-seq] = "-"+ str

		nseq := seq
		if seq == def_seq {
			nseq = -nseq
		}
		bi.STRING2SEQ[     str] =  nseq
		bi.STRING2SEQ["-"+ str] = -nseq
	}
	return bi
}
