package micro_strategy

type KdjIdx struct {
	n1 int
	n2 int
	n3 int
}

// NewKdj(9, 3, 3)
func NewKdj(n1 int, n2 int, n3 int) *KdjIdx {
	return &KdjIdx{n1: n1, n2: n2, n3: n3}
}

func (this *KdjIdx) maxHigh(highs []float64) (h float64) {
	h = highs[0]
	for i := 0; i < len(highs); i++ {
		if highs[i] > h {
			h = highs[i]
		}
	}
	return
}

func (this *KdjIdx) minLow(lows []float64) (l float64) {
	l = lows[0]
	for i := 0; i < len(lows); i++ {
		if lows[i] < l {
			l = lows[i]
		}
	}
	return
}

func (this *KdjIdx) sma(x []float64, n float64) (r []float64) {
	r = make([]float64, len(x))
	for i := 0; i < len(x); i++ {
		if i == 0 {
			r[i] = x[i]
		} else {
			r[i] = (1.0*x[i] + (n-1.0)*r[i-1]) / n
		}
	}
	return
}

func (this *KdjIdx) KdjCall(highs []float64, lows []float64, closes []float64) (k, d, j []float64) {
	l := len(highs)
	rsv := make([]float64, l)
	j = make([]float64, l)
	rsv[0] = 50.0
	for i := 1; i < l; i++ {
		m := i + 1 - this.n1
		if m < 0 {
			m = 0
		}
		h := this.maxHigh(highs[m : i+1])
		l := this.minLow(lows[m : i+1])
		rsv[i] = (closes[i] - l) * 100.0 / (h - l)
		rsv[i] = rsv[i]
	}
	k = this.sma(rsv, float64(this.n2))
	d = this.sma(k, float64(this.n3))
	for i := 0; i < l; i++ {
		j[i] = 3.0*k[i] - 2.0*d[i]
	}
	return
}
