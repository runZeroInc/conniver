/**
 * Copyright (c) 2022, Xerra Earth Observation Institute
 * See LICENSE.TXT in the root directory of this source tree.
 */

package linux

/*
#include "mock_tcpinfo.h"
*/
import "C"
import (
	"unsafe"
)

func (packed *RawTCPInfo) MockSetFields(
	SndWScale uint8,
	RcvWScale uint8,
	DeliveryRateAppLimited bool,
	FastOpenClientFail uint8,
) {
	C.set_fields(unsafe.Pointer(packed),
		C.uchar(SndWScale),
		C.uchar(RcvWScale),
		C.bool(DeliveryRateAppLimited),
		C.uchar(FastOpenClientFail),
	)
}
