package pkg

type Anomalizer struct {
	counter          map[string]int
	limitCounterKeys int
}

func NewAnomalizer() *Anomalizer {
	return &Anomalizer{
		counter:          make(map[string]int),
		limitCounterKeys: 100,
	}
}

func (a *Anomalizer) MemSafeCount(key string) {
	tk := Truncate(key, 100)
	a.counter[tk]++
	if a.counter[tk] > a.limitCounterKeys {
		delete(a.counter, tk)
	}
}
