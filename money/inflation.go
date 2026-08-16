package money

import "math/big"

type Factor struct {
	grown *big.Rat
}

func NoInflation() Factor {
	return Factor{grown: new(big.Rat).SetInt64(1)}
}

func (f Factor) IsOne() bool { return f.value().Cmp(oneRat) == 0 }

var oneRat = new(big.Rat).SetInt64(1)

func (f Factor) After(r Rate) Factor {
	next := new(big.Rat).SetFrac64(r.num+r.den, r.den)
	return Factor{grown: next.Mul(f.value(), next)}
}

func (f Factor) Compose(g Factor) Factor {
	return Factor{grown: new(big.Rat).Mul(f.value(), g.value())}
}

func (f Factor) Apply(y Yen) Yen {
	moved := new(big.Rat).SetInt64(int64(y))
	return Yen(halfUpBig(moved.Mul(moved, f.value())).Int64())
}

func (f Factor) Deflate(y Yen) Yen {
	moved := new(big.Rat).SetInt64(int64(y))
	return Yen(halfUpBig(moved.Quo(moved, f.value())).Int64())
}

func (f Factor) String() string { return f.value().FloatString(4) }

func (f Factor) value() *big.Rat {
	if f.grown == nil {
		return new(big.Rat).SetInt64(1)
	}
	return f.grown
}

func halfUpBig(r *big.Rat) *big.Int {
	n, d := r.Num(), r.Denom()

	twiceD := new(big.Int).Lsh(d, 1)
	scaled := new(big.Int).Lsh(n, 1)

	if n.Sign() >= 0 {
		scaled.Add(scaled, d)
		return scaled.Quo(scaled, twiceD)
	}

	scaled.Neg(scaled).Add(scaled, d)
	return scaled.Quo(scaled, twiceD).Neg(scaled)
}
