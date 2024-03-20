package app

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/rs/zerolog/log"
	"grono.dev/zte-5g/pkg/zte"
)

func Run(ctx context.Context) error {
	for {
		err := run(ctx)
		if !errors.Is(err, zte.ErrSessionExpired) {
			return err
		}
	}
}

var influx = sync.OnceValue(func() api.WriteAPIBlocking {
	token := os.Getenv("INFLUXDB_TOKEN")
	url := os.Getenv("INFLUXDB_URL")
	return influxdb2.NewClient(url, token).WriteAPIBlocking("pawel", "pawel")
})

func run(ctx context.Context) error {
	zteUrl, pass, err := getCredentials()
	if err != nil {
		return err
	}

	z, err := zte.Connect(zteUrl, pass)
	if err != nil {
		return fmt.Errorf("failed to connect to ZTE: %w", err)
	}
	defer z.Close()

	ticker := time.NewTicker(time.Second * 60)
	defer ticker.Stop()

	for {
		if err := update(ctx, z); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func update(ctx context.Context, z *zte.Session) error {
	if err := check5G(ctx, z); err != nil {
		return fmt.Errorf("failed to check 5G: %w", err)
	}
	return nil
}

var lastBand *string

type Status struct {
	Band5G string `json:"nr5g_action_band"`
	Type   string `json:"network_type"`

	RtTxThroughput string `json:"realtime_tx_thrpt"`
	RtRxThroughput string `json:"realtime_rx_thrpt"`

	RtRxBytes string `json:"realtime_rx_bytes"`
	RtTxBytes string `json:"realtime_tx_bytes"`
	RtTime    string `json:"realtime_time"`

	MonthlyRxBytes string `json:"monthly_rx_bytes"`
	MonthlyTxBytes string `json:"monthly_tx_bytes"`
	MonthlyTime    string `json:"monthly_time"`
}

func check5G(ctx context.Context, z *zte.Session) error {
	var status Status
	if err := z.GetCmd(&status); err != nil {
		return err
	}

	log.Info().Str("band5g", status.Band5G).Msgf("connected to band %s", status.Band5G)

	influxStatus(ctx, &status)

	b := status.Band5G
	if lastBand == nil {
		lastBand = &b
		return nil
	}

	if *lastBand == b {
		return nil
	}
	lastBand = &b

	if b == "n78" {
		log.Error().Msg("connected to 5G N78")
	} else if strings.HasPrefix(b, "n") {
		log.Error().Msg("connected to wrong 5G band " + b)
	} else if status.Band5G == "" {
		log.Error().Msg("not connected to 5G")
	}

	return nil
}

func influxStatus(ctx context.Context, status *Status) {
	toInt := func(s string) int64 {
		i, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return -1
		}
		return i
	}

	const mb = 1000 * 1000
	const mib = 1024 * 1024
	const gib = mib * 1024
	const day = 24 * 60 * 60

	rtime := toInt(status.RtTime)
	rtRx := toInt(status.RtRxBytes) * 8 / mb
	rtTx := toInt(status.RtTxBytes) * 8 / mb

	longTime := toInt(status.MonthlyTime)

	ts := time.Now()
	data := map[string]interface{}{
		"type":        status.Type,
		"band":        0,
		"rt_rx_bytes": toInt(status.RtRxBytes),
		"rt_tx_bytes": toInt(status.RtTxBytes),

		"rt_rx_mbit": rtRx,
		"rt_tx_mbit": rtTx,
		"rt_time":    rtime,

		"rt_rx_mbit_ps": rtRx / rtime,
		"rt_tx_mbit_ps": rtTx / rtime,

		"bandwidth_rx": toInt(status.RtRxThroughput),
		"bandwidth_tx": toInt(status.RtTxThroughput),

		"monthly_rx_bytes": toInt(status.MonthlyRxBytes),
		"monthly_tx_bytes": toInt(status.MonthlyTxBytes),
		"monthly_time":     longTime,

		"daily_rx_gib": float64(toInt(status.MonthlyRxBytes)*day) / float64(gib) / float64(longTime),
		"daily_tx_gib": float64(toInt(status.MonthlyTxBytes)*day) / float64(gib) / float64(longTime),
	}

	if status.Band5G != "" {
		bn, err := strconv.Atoi(strings.TrimPrefix(status.Band5G, "n"))
		if err != nil {
			log.Error().Err(err).Msg("failed to parse band number")
			data["band"] = -1
		} else {
			data["band"] = bn
		}
	}
	point := influxdb2.NewPoint("zte", nil, data, ts)
	if err := influx().WritePoint(ctx, point); err != nil {
		log.Error().Err(err).Msg("failed to write influxdb point")
	}
}

func getCredentials() (url.URL, string, error) {
	hostUrlRaw := os.Getenv("ZTE_URL")
	var zteUrl url.URL
	if hostUrlRaw == "" {
		zteUrl = url.URL{Scheme: "http", Host: "192.168.0.1"}
	} else {
		u, err := url.Parse(hostUrlRaw)
		if err != nil {
			return url.URL{}, "", fmt.Errorf("invalid ZTE_URL: %w", err)
		}
		zteUrl = *u
	}
	pass := os.Getenv("ZTE_PASS")
	if pass == "" {
		return url.URL{}, "", fmt.Errorf("ZTE_PASS is empty")
	}
	return zteUrl, pass, nil
}
