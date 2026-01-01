/**
 * Copyright (c) 2022, Xerra Earth Observation Institute
 * See LICENSE.TXT in the root directory of this source tree.
 */

#include "mock_tcpinfo.h"

void set_fields( void* ptr,
                 uint8_t snd_wscale,
                 uint8_t rcv_wscale,
                 bool delivery_rate_app_limited,
                 uint8_t fastopen_client_fail
                 ) {

    struct tcp_info t;
    memset(&t, 0, sizeof(struct tcp_info));

    t.tcpi_snd_wscale = snd_wscale;
    t.tcpi_rcv_wscale = rcv_wscale;
    t.tcpi_delivery_rate_app_limited = delivery_rate_app_limited ? 1 : 0;
    t.tcpi_fastopen_client_fail = fastopen_client_fail;

    memcpy(ptr, &t, sizeof(struct tcp_info));
}