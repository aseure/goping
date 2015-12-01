package tsdb

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	utils_json "github.com/aseure/goping/utils/json"
	"github.com/influxdb/influxdb/client/v2"
)

// Connector to an InfluxDB instance to handle pings as timeseries.
type InfluxConnector struct {
	c        client.Client
	database string
}

// Instantiates a new InfluxConnector and connects to the InfluxDB instance.
func NewInfluxConnector() *InfluxConnector {
	influxClient, err := client.NewHTTPClient(client.HTTPConfig{
		Addr: "http://localhost:8086",
	})

	if err != nil {
		return nil
	}

	return &InfluxConnector{
		c:        influxClient,
		database: "goping",
	}
}

func (connector *InfluxConnector) AddPings(pings []utils_json.Ping) {
	bp, _ := client.NewBatchPoints(client.BatchPointsConfig{
		Database:  connector.database,
		Precision: "s",
	})

	for _, p := range pings {
		tags := map[string]string{
			"origin": p.Origin,
		}

		fields := map[string]interface{}{
			"name_lookup_time_ms": p.NameLookupTimeMs,
			"connect_time_ms":     p.ConnectTimeMs,
			"transfer_time_ms":    p.TransferTimeMs,
			"total_time_ms":       p.TotalTimeMs,
			"status":              p.Status,
		}

		timestamp, err := time.Parse("2006-01-02 15:04:05 MST", p.CreatedAt)
		if err != nil {
			log.Println(err)
		}

		point, err := client.NewPoint(
			"ping",
			tags,
			fields,
			timestamp,
		)

		bp.AddPoint(point)
	}

	if err := connector.c.Write(bp); err != nil {
		log.Println("Cannot add the datapoint to InfluxDB.")
	}
}

func (connector *InfluxConnector) GetAveragePerHour(origin string) []float64 {
	start := connector.findOldestTimestamp(origin)
	nbHours := int(time.Since(start).Hours())

	return connector.getAverages(origin, start, time.Hour, nbHours)
}

// Generic method to retrieve any array of averages.
// For instance, if we need to retrieve averages per hour of the last 24 hours,
// the parameters must be set to:
//
//   - start: time.Now().AddDate(0, 0, -1)
//   - step: time.Hour
//   - count: 24
//
func (connector *InfluxConnector) getAverages(
	origin string,
	start time.Time,
	step time.Duration,
	count int) []float64 {

	averages := make([]float64, count)
	startUnix := start.Unix() * 1000000000
	stepUnix := int64(step.Seconds()) * 1000000000

	for i := 0; i < count; i++ {
		query := fmt.Sprintf(
			"SELECT MEAN(transfer_time_ms) FROM ping WHERE origin = '%s' AND time > %d AND time < %d",
			origin,
			startUnix,
			startUnix+stepUnix,
		)

		res, err := connector.query(query)
		if err != nil {
			log.Println("ERROR")
		}

		averages[i] = 0
		if len(res[0].Series) != 0 {
			if meanItf := res[0].Series[0].Values[0][1]; meanItf != nil {
				if f, err := meanItf.(json.Number).Float64(); err == nil {
					averages[i] = f
				}
			}
		}

		startUnix = startUnix + stepUnix
	}

	return averages
}

// Finds the oldest timestamp for the specified origin
func (connector *InfluxConnector) findOldestTimestamp(origin string) time.Time {
	query := fmt.Sprintf("SELECT status FROM ping WHERE origin = '%s'",
		origin,
	)

	res, err := connector.query(query)
	if err != nil {
		log.Println("ERROR")
	}

	if len(res[0].Series) != 0 {
		if tItf := res[0].Series[0].Values[0][0]; tItf != nil {
			if t, err := time.Parse(time.RFC3339, tItf.(string)); err == nil {
				return t
			}
		}
	}

	return time.Now()
}

// Query wrapper for InfluxDB commands.
func (connector *InfluxConnector) query(cmd string) (res []client.Result, err error) {
	q := client.Query{
		Command:  cmd,
		Database: connector.database,
	}

	if response, err := connector.c.Query(q); err == nil {
		if response.Error() != nil {
			return res, response.Error()
		}
		res = response.Results
	}

	return res, nil
}