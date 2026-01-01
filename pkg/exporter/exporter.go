/**
 * Copyright (c) 2022, Xerra Earth Observation Institute
 * See LICENSE.TXT in the root directory of this source tree.
 */

package exporter

import (
	"fmt"
	"github.com/higebu/netfd"
	"net"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"gitlab.com/xerra/common/go-tcpinfo/pkg/linux"
)

type info struct {
	description *prometheus.Desc
	supplier    func(tcpInfo *linux.TCPInfo, labelValues []string) prometheus.Metric
}

type connEntry struct {
	fd     int
	labels []string
}

type TCPInfoCollector struct {
	conns  map[net.Conn]connEntry
	mu     sync.Mutex
	logger func(error)
	infos  []info
}

func (t *TCPInfoCollector) Describe(descs chan<- *prometheus.Desc) {
	for _, info := range t.infos {
		descs <- info.description
	}
}

func (t *TCPInfoCollector) Collect(metrics chan<- prometheus.Metric) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for conn, entry := range t.conns {
		tcpInfo, err := linux.GetTCPInfo(entry.fd)
		if err != nil {
			t.logger(fmt.Errorf("error getting connection tcpinfo (removing conn %v -> %v): %w", conn.LocalAddr(), conn.RemoteAddr(), err))

			delete(t.conns, conn)
			continue
		}

		for _, info := range t.infos {
			metrics <- info.supplier(tcpInfo, entry.labels)
		}
	}
}

func (t *TCPInfoCollector) Add(conn net.Conn, labels []string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.conns[conn] = connEntry{
		fd:     netfd.GetFdFromConn(conn),
		labels: labels,
	}
}

func (t *TCPInfoCollector) Remove(conn net.Conn) {
	t.mu.Lock()
	defer t.mu.Unlock()

	delete(t.conns, conn)
}

func makeDescriptions(prefix string, variableLabels []string, constLabels prometheus.Labels) map[string]*prometheus.Desc {
	// Most of these descriptions were adapted from the M-Lab project documentation: https://www.measurementlab.net/tests/tcp-info/
	return map[string]*prometheus.Desc{
		"state":                     prometheus.NewDesc(fmt.Sprintf("%s_state", prefix), "Connection state, see include/net/tcp_states.h.", variableLabels, constLabels),
		"ca_state":                  prometheus.NewDesc(fmt.Sprintf("%s_ca_state", prefix), "Loss recovery state machine, see include/net/tcp.h.", variableLabels, constLabels),
		"retransmits":               prometheus.NewDesc(fmt.Sprintf("%s_retransmits", prefix), "Number of timeouts (RTO based retransmissions) at this sequence (reset to zero on forward progress).", variableLabels, constLabels),
		"probes":                    prometheus.NewDesc(fmt.Sprintf("%s_probes", prefix), "Consecutive zero window probes that have gone unanswered.", variableLabels, constLabels),
		"backoff":                   prometheus.NewDesc(fmt.Sprintf("%s_backoff", prefix), "Exponential timeout backoff counter. Increment on RTO, reset on successful RTT measurements.", variableLabels, constLabels),
		"options":                   prometheus.NewDesc(fmt.Sprintf("%s_options", prefix), "Bit encoded SYN options and other negotiations: TIMESTAMPS 0x1; SACK 0x2; WSCALE 0x4; ECN 0x8 - Was negotiated; ECN_SEEN - At least one ECT seen; SYN_DATA - SYN-ACK acknowledged data in SYN sent or rcvd.", variableLabels, constLabels),
		"snd_wscale":                prometheus.NewDesc(fmt.Sprintf("%s_snd_wscale", prefix), "Window scaling of send-half of connection (bit shift).", variableLabels, constLabels),
		"rcv_wscale":                prometheus.NewDesc(fmt.Sprintf("%s_rcv_wscale", prefix), "Window scaling of receive-half of connection (bit shift).", variableLabels, constLabels),
		"delivery_rate_app_limited": prometheus.NewDesc(fmt.Sprintf("%s_delivery_rate_app_limited", prefix), "Flag indicating that rate measurements reflect non-network bottlenecks (1.0 = true, 0.0 = false)", variableLabels, constLabels),
		"fastopen_client_fail":      prometheus.NewDesc(fmt.Sprintf("%s_fastopen_client_fail", prefix), "The reason why TCP fastopen failed. 0x0: unspecified; 0x1: no cookie sent; 0x2: SYN-ACK did not ack SYN data; 0x3: SYN-ACK did not ack SYN data after timeout (-1.0 if unavailable).", variableLabels, constLabels),
		"rto":                       prometheus.NewDesc(fmt.Sprintf("%s_rto", prefix), "Retransmission Timeout. Quantized to system jiffies.", variableLabels, constLabels),
		"ato":                       prometheus.NewDesc(fmt.Sprintf("%s_ato", prefix), "Delayed ACK Timeout. Quantized to system jiffies.", variableLabels, constLabels),
		"snd_mss":                   prometheus.NewDesc(fmt.Sprintf("%s_snd_mss", prefix), "Current Maximum Segment Size. Note that this can be smaller than the negotiated MSS for various reasons.", variableLabels, constLabels),
		"rcv_mss":                   prometheus.NewDesc(fmt.Sprintf("%s_rcv_mss", prefix), "Maximum observed segment size from the remote host. Used to trigger delayed ACKs.", variableLabels, constLabels),
		"unacked":                   prometheus.NewDesc(fmt.Sprintf("%s_unacked", prefix), "Number of segments between snd.nxt and snd.una. Accounting for the Pipe algorithm.", variableLabels, constLabels),
		"sacked":                    prometheus.NewDesc(fmt.Sprintf("%s_sacked", prefix), "Scoreboard segment marked SACKED by sack blocks. Accounting for the Pipe algorithm.", variableLabels, constLabels),
		"lost":                      prometheus.NewDesc(fmt.Sprintf("%s_lost", prefix), "Scoreboard segments marked lost by loss detection heuristics. Accounting for the Pipe algorithm.", variableLabels, constLabels),
		"retrans":                   prometheus.NewDesc(fmt.Sprintf("%s_retrans", prefix), "Scoreboard segments marked retransmitted. Accounting for the Pipe algorithm.", variableLabels, constLabels),
		"fackets":                   prometheus.NewDesc(fmt.Sprintf("%s_fackets", prefix), "Some counter in Forward Acknowledgment (FACK) TCP congestion control. M-Lab says this is unused?", variableLabels, constLabels),
		"last_data_sent":            prometheus.NewDesc(fmt.Sprintf("%s_last_data_sent", prefix), "Time since last data segment was sent. Quantized to jiffies.", variableLabels, constLabels),
		"last_ack_sent":             prometheus.NewDesc(fmt.Sprintf("%s_last_ack_sent", prefix), "Time since last ACK was sent. Not implemented!", variableLabels, constLabels),
		"last_data_recv":            prometheus.NewDesc(fmt.Sprintf("%s_last_data_recv", prefix), "Time since last data segment was received. Quantized to jiffies.", variableLabels, constLabels),
		"last_ack_recv":             prometheus.NewDesc(fmt.Sprintf("%s_last_ack_recv", prefix), "Time since last ACK was received. Quantized to jiffies.", variableLabels, constLabels),
		"pmtu":                      prometheus.NewDesc(fmt.Sprintf("%s_pmtu", prefix), "Maximum IP Transmission Unit for this path.", variableLabels, constLabels),
		"rcv_ssthresh":              prometheus.NewDesc(fmt.Sprintf("%s_rcv_ssthresh", prefix), "Current Window Clamp. Receiver algorithm to avoid allocating excessive receive buffers.", variableLabels, constLabels),
		"rtt":                       prometheus.NewDesc(fmt.Sprintf("%s_rtt", prefix), "Smoothed Round Trip Time (RTT). The Linux implementation differs from the standard.", variableLabels, constLabels),
		"rttvar":                    prometheus.NewDesc(fmt.Sprintf("%s_rttvar", prefix), "RTT variance. The Linux implementation differs from the standard.", variableLabels, constLabels),
		"snd_ssthresh":              prometheus.NewDesc(fmt.Sprintf("%s_snd_ssthresh", prefix), "Slow Start Threshold. Value controlled by the selected congestion control algorithm.", variableLabels, constLabels),
		"snd_cwnd":                  prometheus.NewDesc(fmt.Sprintf("%s_snd_cwnd", prefix), "Congestion Window. Value controlled by the selected congestion control algorithm.", variableLabels, constLabels),
		"advmss":                    prometheus.NewDesc(fmt.Sprintf("%s_advmss", prefix), "Advertised maximum segment size.", variableLabels, constLabels),
		"reordering":                prometheus.NewDesc(fmt.Sprintf("%s_reordering", prefix), "Maximum observed reordering distance.", variableLabels, constLabels),
		"rcv_rtt":                   prometheus.NewDesc(fmt.Sprintf("%s_rcv_rtt", prefix), "Receiver Side RTT estimate.", variableLabels, constLabels),
		"rcv_space":                 prometheus.NewDesc(fmt.Sprintf("%s_rcv_space", prefix), "Space reserved for the receive queue. Typically updated by receiver side auto-tuning.", variableLabels, constLabels),
		"total_retrans":             prometheus.NewDesc(fmt.Sprintf("%s_total_retrans", prefix), "Total number of segments containing retransmitted data.", variableLabels, constLabels),
		"pacing_rate":               prometheus.NewDesc(fmt.Sprintf("%s_pacing_rate", prefix), "Current Pacing Rate, nominally updated by congestion control.", variableLabels, constLabels),
		"max_pacing_rate":           prometheus.NewDesc(fmt.Sprintf("%s_max_pacing_rate", prefix), "Settable pacing rate clamp. Set with setsockopt( ..SO_MAX_PACING_RATE.. ).", variableLabels, constLabels),
		"bytes_acked":               prometheus.NewDesc(fmt.Sprintf("%s_bytes_acked", prefix), "The number of data bytes for which cumulative acknowledgments have been received | RFC4898 tcpEStatsAppHCThruOctetsAcked.", variableLabels, constLabels),
		"bytes_received":            prometheus.NewDesc(fmt.Sprintf("%s_bytes_received", prefix), "The number of data bytes for which cumulative acknowledgments have been sent | RFC4898 tcpEStatsAppHCThruOctetsReceived.", variableLabels, constLabels),
		"segs_out":                  prometheus.NewDesc(fmt.Sprintf("%s_segs_out", prefix), "The number of segments transmitted. Includes data and pure ACKs | RFC4898 tcpEStatsPerfSegsOut.", variableLabels, constLabels),
		"segs_in":                   prometheus.NewDesc(fmt.Sprintf("%s_segs_in", prefix), "The number of segments received. Includes data and pure ACKs | RFC4898 tcpEStatsPerfSegsIn.", variableLabels, constLabels),
		"notsent_bytes":             prometheus.NewDesc(fmt.Sprintf("%s_notsent_bytes", prefix), "Number of bytes queued in the send buffer that have not been sent.", variableLabels, constLabels),
		"min_rtt":                   prometheus.NewDesc(fmt.Sprintf("%s_min_rtt", prefix), "Minimum RTT. From an older, pre-BBR algorithm.", variableLabels, constLabels),
		"data_segs_in":              prometheus.NewDesc(fmt.Sprintf("%s_data_segs_in", prefix), "Input segments carrying data (len>0) | RFC4898 tcpEStatsDataSegsIn (actually tcpEStatsPerfDataSegsIn).", variableLabels, constLabels),
		"data_segs_out":             prometheus.NewDesc(fmt.Sprintf("%s_data_segs_out", prefix), "Transmitted segments carrying data (len>0) | RFC4898 tcpEStatsDataSegsOut (actually tcpEStatsPerfDataSegsOut).", variableLabels, constLabels),
		"delivery_rate":             prometheus.NewDesc(fmt.Sprintf("%s_delivery_rate", prefix), "Observed Maximum Delivery Rate", variableLabels, constLabels),
		"busy_time":                 prometheus.NewDesc(fmt.Sprintf("%s_busy_time", prefix), "Time in usecs with outstanding (unacknowledged) data. Time when snd.una not equal to snd.next.", variableLabels, constLabels),
		"rwnd_limited":              prometheus.NewDesc(fmt.Sprintf("%s_rwnd_limited", prefix), "Time in usecs spent limited by/waiting for receiver window.", variableLabels, constLabels),
		"sndbuf_limited":            prometheus.NewDesc(fmt.Sprintf("%s_sndbuf_limited", prefix), "Time in usecs spent limited by/waiting for sender buffer space. This only includes the time when TCP transmissions are starved for data, but the application has been stopped because the buffer is full and can not be grown for some reason.", variableLabels, constLabels),
		"delivered":                 prometheus.NewDesc(fmt.Sprintf("%s_delivered", prefix), "Data segments delivered to the receiver including retransmits. As reported by returning ACKs, used by ECN.", variableLabels, constLabels),
		"delivered_ce":              prometheus.NewDesc(fmt.Sprintf("%s_delivered_ce", prefix), "ECE marked data segments delivered to the receiver including retransmits. As reported by returning ACKs, used by ECN.", variableLabels, constLabels),
		"bytes_sent":                prometheus.NewDesc(fmt.Sprintf("%s_bytes_sent", prefix), "Payload bytes sent (excludes headers, includes retransmissions) | RFC4898 tcpEStatsPerfHCDataOctetsOut.", variableLabels, constLabels),
		"bytes_retrans":             prometheus.NewDesc(fmt.Sprintf("%s_bytes_retrans", prefix), "Bytes retransmitted. May include headers and new data carried with a retransmission (for thin flows) | RFC4898 tcpEStatsPerfOctetsRetrans.", variableLabels, constLabels),
		"dsack_dups":                prometheus.NewDesc(fmt.Sprintf("%s_dsack_dups", prefix), "Duplicate segments reported by DSACK | RFC4898 tcpEStatsStackDSACKDups.", variableLabels, constLabels),
		"reord_seen":                prometheus.NewDesc(fmt.Sprintf("%s_reord_seen", prefix), "Received ACKs that were out of order. Estimates reordering on the return path.", variableLabels, constLabels),
		"rcv_ooopack":               prometheus.NewDesc(fmt.Sprintf("%s_rcv_ooopack", prefix), "Out-of-order packets received.", variableLabels, constLabels),
		"snd_wnd":                   prometheus.NewDesc(fmt.Sprintf("%s_snd_wnd", prefix), "Peer's advertised receive window after scaling (bytes).", variableLabels, constLabels),
	}
}

func NewTCPInfoCollector(
	prefix string,
	connectionLabels []string, // connectionLabels are known up front for the collector and values are provided when adding a connection.
	constLabels prometheus.Labels, // constLabels is meant for labels with values that are constant for the whole process.
	errorLoggingCallback func(error),
) TCPInfoCollector {
	desc := makeDescriptions(prefix, connectionLabels, constLabels)

	infos := []info{
		{description: desc["state"], supplier: func(tcpInfo *linux.TCPInfo, labelValues []string) prometheus.Metric { // Empirically a gauge
			return prometheus.MustNewConstMetric(desc["state"], prometheus.GaugeValue, float64(tcpInfo.State), labelValues...)
		}},
		{description: desc["ca_state"], supplier: func(tcpInfo *linux.TCPInfo, labelValues []string) prometheus.Metric { // Empirically a gauge
			return prometheus.MustNewConstMetric(desc["ca_state"], prometheus.GaugeValue, float64(tcpInfo.CAState), labelValues...)
		}},
		{description: desc["retransmits"], supplier: func(tcpInfo *linux.TCPInfo, labelValues []string) prometheus.Metric { // Empirically a gauge
			return prometheus.MustNewConstMetric(desc["retransmits"], prometheus.GaugeValue, float64(tcpInfo.Retransmits), labelValues...)
		}},
		{description: desc["probes"], supplier: func(tcpInfo *linux.TCPInfo, labelValues []string) prometheus.Metric { // // Empirically a gauge
			return prometheus.MustNewConstMetric(desc["probes"], prometheus.GaugeValue, float64(tcpInfo.Probes), labelValues...)
		}},
		{description: desc["backoff"], supplier: func(tcpInfo *linux.TCPInfo, labelValues []string) prometheus.Metric { // Empirically a gauge
			return prometheus.MustNewConstMetric(desc["backoff"], prometheus.GaugeValue, float64(tcpInfo.Backoff), labelValues...)
		}},
		{description: desc["options"], supplier: func(tcpInfo *linux.TCPInfo, labelValues []string) prometheus.Metric { // Bitmap so has to be a gauge
			return prometheus.MustNewConstMetric(desc["options"], prometheus.GaugeValue, float64(tcpInfo.Options), labelValues...)
		}},
		{description: desc["snd_wscale"], supplier: func(tcpInfo *linux.TCPInfo, labelValues []string) prometheus.Metric { // Presumably a gauge
			return prometheus.MustNewConstMetric(desc["snd_wscale"], prometheus.GaugeValue, float64(tcpInfo.SndWScale), labelValues...)
		}},
		{description: desc["rcv_wscale"], supplier: func(tcpInfo *linux.TCPInfo, labelValues []string) prometheus.Metric { // Presumably a gauge
			return prometheus.MustNewConstMetric(desc["rcv_wscale"], prometheus.GaugeValue, float64(tcpInfo.RcvWScale), labelValues...)
		}},
		{description: desc["delivery_rate_app_limited"], supplier: func(tcpInfo *linux.TCPInfo, labelValues []string) prometheus.Metric { // Bool so has to be a gauge
			val := 0.0
			if tcpInfo.DeliveryRateAppLimited {
				val = 1.0
			}
			return prometheus.MustNewConstMetric(desc["delivery_rate_app_limited"], prometheus.GaugeValue, val, labelValues...)
		}},
		{description: desc["rto"], supplier: func(tcpInfo *linux.TCPInfo, labelValues []string) prometheus.Metric { // Empirically a gauge
			return prometheus.MustNewConstMetric(desc["rto"], prometheus.GaugeValue, float64(tcpInfo.RTO), labelValues...)
		}},
		{description: desc["ato"], supplier: func(tcpInfo *linux.TCPInfo, labelValues []string) prometheus.Metric { // Empirically a gauge
			return prometheus.MustNewConstMetric(desc["ato"], prometheus.GaugeValue, float64(tcpInfo.ATO), labelValues...)
		}},
		{description: desc["snd_mss"], supplier: func(tcpInfo *linux.TCPInfo, labelValues []string) prometheus.Metric { // Presumably a gauge
			return prometheus.MustNewConstMetric(desc["snd_mss"], prometheus.GaugeValue, float64(tcpInfo.SndMSS), labelValues...)
		}},
		{description: desc["rcv_mss"], supplier: func(tcpInfo *linux.TCPInfo, labelValues []string) prometheus.Metric { // Empirically a gauge
			return prometheus.MustNewConstMetric(desc["rcv_mss"], prometheus.GaugeValue, float64(tcpInfo.RcvMSS), labelValues...)
		}},
		{description: desc["unacked"], supplier: func(tcpInfo *linux.TCPInfo, labelValues []string) prometheus.Metric { // Empirically a gauge
			return prometheus.MustNewConstMetric(desc["unacked"], prometheus.GaugeValue, float64(tcpInfo.UnAcked), labelValues...)
		}},
		{description: desc["sacked"], supplier: func(tcpInfo *linux.TCPInfo, labelValues []string) prometheus.Metric { // Empirically a gauge
			return prometheus.MustNewConstMetric(desc["sacked"], prometheus.GaugeValue, float64(tcpInfo.Sacked), labelValues...)
		}},
		{description: desc["lost"], supplier: func(tcpInfo *linux.TCPInfo, labelValues []string) prometheus.Metric { // Empirically a gauge
			return prometheus.MustNewConstMetric(desc["lost"], prometheus.GaugeValue, float64(tcpInfo.Lost), labelValues...)
		}},
		{description: desc["retrans"], supplier: func(tcpInfo *linux.TCPInfo, labelValues []string) prometheus.Metric { // Empirically a gauge
			return prometheus.MustNewConstMetric(desc["retrans"], prometheus.GaugeValue, float64(tcpInfo.Retrans), labelValues...)
		}},
		{description: desc["fackets"], supplier: func(tcpInfo *linux.TCPInfo, labelValues []string) prometheus.Metric { // Presumably a counter
			return prometheus.MustNewConstMetric(desc["fackets"], prometheus.CounterValue, float64(tcpInfo.Fackets), labelValues...)
		}},
		{description: desc["last_data_sent"], supplier: func(tcpInfo *linux.TCPInfo, labelValues []string) prometheus.Metric { // Empirically a gauge
			return prometheus.MustNewConstMetric(desc["last_data_sent"], prometheus.GaugeValue, float64(tcpInfo.LastDataSent), labelValues...)
		}},
		{description: desc["last_ack_sent"], supplier: func(tcpInfo *linux.TCPInfo, labelValues []string) prometheus.Metric { // Presumably a gauge if it was implemented
			return prometheus.MustNewConstMetric(desc["last_ack_sent"], prometheus.GaugeValue, float64(tcpInfo.LastAckSent), labelValues...)
		}},
		{description: desc["last_data_recv"], supplier: func(tcpInfo *linux.TCPInfo, labelValues []string) prometheus.Metric { // Empirically a gauge
			return prometheus.MustNewConstMetric(desc["last_data_recv"], prometheus.GaugeValue, float64(tcpInfo.LastDataRecv), labelValues...)
		}},
		{description: desc["last_ack_recv"], supplier: func(tcpInfo *linux.TCPInfo, labelValues []string) prometheus.Metric { // Empirically a gauge
			return prometheus.MustNewConstMetric(desc["last_ack_recv"], prometheus.GaugeValue, float64(tcpInfo.LastAckRecv), labelValues...)
		}},
		{description: desc["pmtu"], supplier: func(tcpInfo *linux.TCPInfo, labelValues []string) prometheus.Metric { // Presumably a gauge (PMDU may be re-probed or a message may trigger decrease)
			return prometheus.MustNewConstMetric(desc["pmtu"], prometheus.GaugeValue, float64(tcpInfo.PMTU), labelValues...)
		}},
		{description: desc["rcv_ssthresh"], supplier: func(tcpInfo *linux.TCPInfo, labelValues []string) prometheus.Metric { // Presumably a gauge
			return prometheus.MustNewConstMetric(desc["rcv_ssthresh"], prometheus.GaugeValue, float64(tcpInfo.RcvSSThresh), labelValues...)
		}},
		{description: desc["rtt"], supplier: func(tcpInfo *linux.TCPInfo, labelValues []string) prometheus.Metric { // Empirically a gauge
			return prometheus.MustNewConstMetric(desc["rtt"], prometheus.GaugeValue, float64(tcpInfo.RTT), labelValues...)
		}},
		{description: desc["rttvar"], supplier: func(tcpInfo *linux.TCPInfo, labelValues []string) prometheus.Metric { // Empirically a gauge
			return prometheus.MustNewConstMetric(desc["rttvar"], prometheus.GaugeValue, float64(tcpInfo.RTTVar), labelValues...)
		}},
		{description: desc["snd_ssthresh"], supplier: func(tcpInfo *linux.TCPInfo, labelValues []string) prometheus.Metric { // Empirically a gauge
			return prometheus.MustNewConstMetric(desc["snd_ssthresh"], prometheus.GaugeValue, float64(tcpInfo.SndSSThresh), labelValues...)
		}},
		{description: desc["snd_cwnd"], supplier: func(tcpInfo *linux.TCPInfo, labelValues []string) prometheus.Metric { // Empirically a gauge
			return prometheus.MustNewConstMetric(desc["snd_cwnd"], prometheus.GaugeValue, float64(tcpInfo.SndCWnd), labelValues...)
		}},
		{description: desc["advmss"], supplier: func(tcpInfo *linux.TCPInfo, labelValues []string) prometheus.Metric { // Presumably a gauge
			return prometheus.MustNewConstMetric(desc["advmss"], prometheus.GaugeValue, float64(tcpInfo.AdvMSS), labelValues...)
		}},
		{description: desc["reordering"], supplier: func(tcpInfo *linux.TCPInfo, labelValues []string) prometheus.Metric { // Presumably a gauge
			return prometheus.MustNewConstMetric(desc["reordering"], prometheus.GaugeValue, float64(tcpInfo.Reordering), labelValues...)
		}},
		{description: desc["rcv_rtt"], supplier: func(tcpInfo *linux.TCPInfo, labelValues []string) prometheus.Metric { // Presumably a gauge
			return prometheus.MustNewConstMetric(desc["rcv_rtt"], prometheus.GaugeValue, float64(tcpInfo.RcvRTT), labelValues...)
		}},
		{description: desc["rcv_space"], supplier: func(tcpInfo *linux.TCPInfo, labelValues []string) prometheus.Metric { // Presumably a gauge
			return prometheus.MustNewConstMetric(desc["rcv_space"], prometheus.GaugeValue, float64(tcpInfo.RcvSpace), labelValues...)
		}},
		{description: desc["total_retrans"], supplier: func(tcpInfo *linux.TCPInfo, labelValues []string) prometheus.Metric { // Empirically a gauge
			return prometheus.MustNewConstMetric(desc["total_retrans"], prometheus.GaugeValue, float64(tcpInfo.TotalRetrans), labelValues...)
		}},
		{description: desc["pacing_rate"], supplier: func(tcpInfo *linux.TCPInfo, labelValues []string) prometheus.Metric { // Empirically a gauge
			return prometheus.MustNewConstMetric(desc["pacing_rate"], prometheus.GaugeValue, float64(tcpInfo.PacingRate), labelValues...)
		}},
		{description: desc["max_pacing_rate"], supplier: func(tcpInfo *linux.TCPInfo, labelValues []string) prometheus.Metric { // Presumably a gauge
			return prometheus.MustNewConstMetric(desc["max_pacing_rate"], prometheus.GaugeValue, float64(tcpInfo.MaxPacingRate), labelValues...)
		}},
		{description: desc["bytes_acked"], supplier: func(tcpInfo *linux.TCPInfo, labelValues []string) prometheus.Metric { // Empirically a gauge even though RFC4898 says it is a counter (!!!)
			return prometheus.MustNewConstMetric(desc["bytes_acked"], prometheus.GaugeValue, float64(tcpInfo.BytesAcked), labelValues...)
		}},
		{description: desc["bytes_received"], supplier: func(tcpInfo *linux.TCPInfo, labelValues []string) prometheus.Metric { // Counter according to MIB
			return prometheus.MustNewConstMetric(desc["bytes_received"], prometheus.CounterValue, float64(tcpInfo.BytesReceived), labelValues...)
		}},
		{description: desc["segs_out"], supplier: func(tcpInfo *linux.TCPInfo, labelValues []string) prometheus.Metric { // Empirically a gauge even though RFC4898 says it is a counter (!!!)
			return prometheus.MustNewConstMetric(desc["segs_out"], prometheus.GaugeValue, float64(tcpInfo.SegsOut), labelValues...)
		}},
		{description: desc["segs_in"], supplier: func(tcpInfo *linux.TCPInfo, labelValues []string) prometheus.Metric { // Empirically a gauge even though RFC4898 says it is a counter (!!!)
			return prometheus.MustNewConstMetric(desc["segs_in"], prometheus.GaugeValue, float64(tcpInfo.SegsIn), labelValues...)
		}},
		{description: desc["notsent_bytes"], supplier: func(tcpInfo *linux.TCPInfo, labelValues []string) prometheus.Metric { // Empirically a gauge
			return prometheus.MustNewConstMetric(desc["notsent_bytes"], prometheus.GaugeValue, float64(tcpInfo.NotsentBytes), labelValues...)
		}},
		{description: desc["min_rtt"], supplier: func(tcpInfo *linux.TCPInfo, labelValues []string) prometheus.Metric { // Empirically a gauge
			return prometheus.MustNewConstMetric(desc["min_rtt"], prometheus.GaugeValue, float64(tcpInfo.MinRTT), labelValues...)
		}},
		{description: desc["data_segs_in"], supplier: func(tcpInfo *linux.TCPInfo, labelValues []string) prometheus.Metric { // Empirically a gauge even though RFC4898 says tcpEStatsPerfDataSegsIn is a counter (!!!)
			return prometheus.MustNewConstMetric(desc["data_segs_in"], prometheus.GaugeValue, float64(tcpInfo.DataSegsIn), labelValues...)
		}},
		{description: desc["data_segs_out"], supplier: func(tcpInfo *linux.TCPInfo, labelValues []string) prometheus.Metric { // Empirically a gauge even though RFC4898 says tcpEStatsPerfDataSegsOut is a counter (!!!)
			return prometheus.MustNewConstMetric(desc["data_segs_out"], prometheus.GaugeValue, float64(tcpInfo.DataSegsOut), labelValues...)
		}},
		{description: desc["delivery_rate"], supplier: func(tcpInfo *linux.TCPInfo, labelValues []string) prometheus.Metric { // Empirically a gauge
			return prometheus.MustNewConstMetric(desc["delivery_rate"], prometheus.GaugeValue, float64(tcpInfo.DeliveryRate), labelValues...)
		}},
		{description: desc["busy_time"], supplier: func(tcpInfo *linux.TCPInfo, labelValues []string) prometheus.Metric { // Empirically a gauge
			return prometheus.MustNewConstMetric(desc["busy_time"], prometheus.GaugeValue, float64(tcpInfo.BusyTime), labelValues...)
		}},
		{description: desc["rwnd_limited"], supplier: func(tcpInfo *linux.TCPInfo, labelValues []string) prometheus.Metric { // Empirically a gauge
			return prometheus.MustNewConstMetric(desc["rwnd_limited"], prometheus.GaugeValue, float64(tcpInfo.RwndLimited), labelValues...)
		}},
		{description: desc["sndbuf_limited"], supplier: func(tcpInfo *linux.TCPInfo, labelValues []string) prometheus.Metric { // Presumably a gauge since RwndLimited is empirically a gauge
			return prometheus.MustNewConstMetric(desc["sndbuf_limited"], prometheus.GaugeValue, float64(tcpInfo.SndbufLimited), labelValues...)
		}},
		{description: desc["delivered"], supplier: func(tcpInfo *linux.TCPInfo, labelValues []string) prometheus.Metric { // Empirically a gauge
			return prometheus.MustNewConstMetric(desc["delivered"], prometheus.GaugeValue, float64(tcpInfo.Delivered), labelValues...)
		}},
		{description: desc["delivered_ce"], supplier: func(tcpInfo *linux.TCPInfo, labelValues []string) prometheus.Metric { // Presumably a gauge since Delivered is empirically a gauge
			return prometheus.MustNewConstMetric(desc["delivered_ce"], prometheus.GaugeValue, float64(tcpInfo.DeliveredCE), labelValues...)
		}},
		{description: desc["bytes_sent"], supplier: func(tcpInfo *linux.TCPInfo, labelValues []string) prometheus.Metric { // Empirically a gauge
			return prometheus.MustNewConstMetric(desc["bytes_sent"], prometheus.GaugeValue, float64(tcpInfo.BytesSent), labelValues...)
		}},
		{description: desc["bytes_retrans"], supplier: func(tcpInfo *linux.TCPInfo, labelValues []string) prometheus.Metric { // Empirically a gauge
			return prometheus.MustNewConstMetric(desc["bytes_retrans"], prometheus.GaugeValue, float64(tcpInfo.BytesRetrans), labelValues...)
		}},
		{description: desc["dsack_dups"], supplier: func(tcpInfo *linux.TCPInfo, labelValues []string) prometheus.Metric { // Empirically a gauge
			return prometheus.MustNewConstMetric(desc["dsack_dups"], prometheus.GaugeValue, float64(tcpInfo.DSACKDups), labelValues...)
		}},
		{description: desc["reord_seen"], supplier: func(tcpInfo *linux.TCPInfo, labelValues []string) prometheus.Metric { // Presumably a counter
			return prometheus.MustNewConstMetric(desc["reord_seen"], prometheus.CounterValue, float64(tcpInfo.ReordSeen), labelValues...)
		}},
		{description: desc["rcv_ooopack"], supplier: func(tcpInfo *linux.TCPInfo, labelValues []string) prometheus.Metric { // Presumably a counter
			return prometheus.MustNewConstMetric(desc["rcv_ooopack"], prometheus.CounterValue, float64(tcpInfo.RcvOOOPack), labelValues...)
		}},
		{description: desc["snd_wnd"], supplier: func(tcpInfo *linux.TCPInfo, labelValues []string) prometheus.Metric { // Empirically a gauge
			return prometheus.MustNewConstMetric(desc["snd_wnd"], prometheus.GaugeValue, float64(tcpInfo.SndWnd), labelValues...)
		}},
	}

	if linux.CheckKernelVersion(5, 5, 0) { // added in v5.5
		infos = append(infos, info{description: desc["fastopen_client_fail"], supplier: func(tcpInfo *linux.TCPInfo, labelValues []string) prometheus.Metric { // Enum so has to be a gauge
			return prometheus.MustNewConstMetric(desc["fastopen_client_fail"], prometheus.GaugeValue, float64(tcpInfo.FastOpenClientFail.Value), labelValues...)
		}})
	}

	return TCPInfoCollector{ //nolint:exhaustivestruct
		conns:  make(map[net.Conn]connEntry),
		logger: errorLoggingCallback,
		infos:  infos,
	}
}
