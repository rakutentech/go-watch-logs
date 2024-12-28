package pkg

import (
	"database/sql"
	"time"
)

type Anomalizer struct {
	f                Flags
	db               *sql.DB
	now              time.Time
	key              string
	window           int
	counters         []Counter
	limitCounterKeys int
}
type Counter struct {
	Match string
	Value int
}

func NewAnomalizer(db *sql.DB, f Flags, now time.Time, key string, window int) *Anomalizer {
	return &Anomalizer{
		db:               db,
		f:                f,
		now:              now,
		key:              key,
		window:           window,
		counters:         []Counter{},
		limitCounterKeys: 100,
	}
}

func (a *Anomalizer) MemSafeCount(match string) {
	// Check if the key exists in counters
	for i, counter := range a.counters {
		if counter.Match == match {
			// Increment the counter if the key exists
			a.counters[i].Value++
			// If the counter exceeds the limit, remove it
			if a.counters[i].Value > a.limitCounterKeys {
				a.counters = append(a.counters[:i], a.counters[i+1:]...)
			}
			return
		}
	}

	// If the key does not exist, add a new counter
	a.counters = append(a.counters, Counter{Match: match, Value: 1})
}

type Anomaly struct {
	Match string
	Value int
}

func (a *Anomalizer) GetAnomalies(match string) ([]int, error) {
	// Query to fetch anomalies within the time range
	rows, err := a.db.Query(
		`SELECT match, value
		FROM anomalies
		WHERE key = ?
		AND match = ?
		AND time = ?`,
		a.key, match, a.now.Format("15:04"),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Iterate over query results and populate the slice
	values := []int{}
	for rows.Next() {
		var anomaly Anomaly
		if err := rows.Scan(&anomaly.Match, &anomaly.Value); err != nil {
			return nil, err
		}
		values = append(values, anomaly.Value)
	}

	return values, nil
}

func (a *Anomalizer) SaveAnomalies() error {
	for _, counter := range a.counters {
		_, err := a.db.Exec(`INSERT INTO anomalies (key, match, value, date, time) VALUES (?, ?, ?, ?, ?)`, a.key, counter.Match, counter.Value, a.now.Format("2006-01-02"), a.now.Format("15:04"))
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *Anomalizer) DeleteOldAnomalies() error {
	windowAt := a.now.AddDate(0, 0, -a.window).Format("2006-01-02")
	_, err := a.db.Exec(`DELETE FROM anomalies WHERE key = ? AND date < ?`, a.key, windowAt)
	return err
}
