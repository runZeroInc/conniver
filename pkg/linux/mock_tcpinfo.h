/**
 * Copyright (c) 2022, Xerra Earth Observation Institute
 *
 * Portions are derived from of Linux's tcp.h, used under the syscall exception
 * (see https://spdx.org/licenses/Linux-syscall-note.html).
 *
 * See LICENSE.TXT in the root directory of this source tree.
 */

#ifndef _MOCK_TCPINFO_H_
#define _MOCK_TCPINFO_H_

#include <stdio.h>
#include <string.h>
#include <stdint.h>
#include <stdbool.h>

struct tcp_info {
	uint8_t	tcpi_state;
	uint8_t	tcpi_ca_state;
	uint8_t	tcpi_retransmits;
	uint8_t	tcpi_probes;
	uint8_t	tcpi_backoff;
	uint8_t	tcpi_options;
	uint8_t	tcpi_snd_wscale : 4, tcpi_rcv_wscale : 4;
	uint8_t	tcpi_delivery_rate_app_limited:1, tcpi_fastopen_client_fail:2;

	uint32_t	tcpi_rto;
	uint32_t	tcpi_ato;
	uint32_t	tcpi_snd_mss;
	uint32_t	tcpi_rcv_mss;

	uint32_t	tcpi_unacked;
	uint32_t	tcpi_sacked;
	uint32_t	tcpi_lost;
	uint32_t	tcpi_retrans;
	uint32_t	tcpi_fackets;

	/* Times. */
	uint32_t	tcpi_last_data_sent;
	uint32_t	tcpi_last_ack_sent;     /* Not remembered, sorry. */
	uint32_t	tcpi_last_data_recv;
	uint32_t	tcpi_last_ack_recv;

	/* Metrics. */
	uint32_t	tcpi_pmtu;
	uint32_t	tcpi_rcv_ssthresh;
	uint32_t	tcpi_rtt;
	uint32_t	tcpi_rttvar;
	uint32_t	tcpi_snd_ssthresh;
	uint32_t	tcpi_snd_cwnd;
	uint32_t	tcpi_advmss;
	uint32_t	tcpi_reordering;

	uint32_t	tcpi_rcv_rtt;
	uint32_t	tcpi_rcv_space;

	uint32_t	tcpi_total_retrans;

	uint64_t	tcpi_pacing_rate;
	uint64_t	tcpi_max_pacing_rate;
	uint64_t	tcpi_bytes_acked;    /* RFC4898 tcpEStatsAppHCThruOctetsAcked */
	uint64_t	tcpi_bytes_received; /* RFC4898 tcpEStatsAppHCThruOctetsReceived */
	uint32_t	tcpi_segs_out;	     /* RFC4898 tcpEStatsPerfSegsOut */
	uint32_t	tcpi_segs_in;	     /* RFC4898 tcpEStatsPerfSegsIn */

	uint32_t	tcpi_notsent_bytes;
	uint32_t	tcpi_min_rtt;
	uint32_t	tcpi_data_segs_in;	/* RFC4898 tcpEStatsDataSegsIn */
	uint32_t	tcpi_data_segs_out;	/* RFC4898 tcpEStatsDataSegsOut */

	uint64_t   tcpi_delivery_rate;

	uint64_t	tcpi_busy_time;      /* Time (usec) busy sending data */
	uint64_t	tcpi_rwnd_limited;   /* Time (usec) limited by receive window */
	uint64_t	tcpi_sndbuf_limited; /* Time (usec) limited by send buffer */

	uint32_t	tcpi_delivered;
	uint32_t	tcpi_delivered_ce;

	uint64_t	tcpi_bytes_sent;     /* RFC4898 tcpEStatsPerfHCDataOctetsOut */
	uint64_t	tcpi_bytes_retrans;  /* RFC4898 tcpEStatsPerfOctetsRetrans */
	uint32_t	tcpi_dsack_dups;     /* RFC4898 tcpEStatsStackDSACKDups */
	uint32_t	tcpi_reord_seen;     /* reordering events seen */

	uint32_t	tcpi_rcv_ooopack;    /* Out-of-order packets received */

	uint32_t	tcpi_snd_wnd;	     /* peer's advertised receive window after
				      * scaling (bytes)
				      */
};

void set_fields(
    void* ptr,
    uint8_t snd_wscale,
    uint8_t rcv_wscale,
    bool delivery_rate_app_limited,
    uint8_t fastopen_client_fail
    );

#endif /* _MOCK_TCPINFO_H_ */