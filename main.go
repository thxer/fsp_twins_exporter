package main

import (
	"encoding/binary"
	"fmt"
	"github.com/go-daq/smbus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"math"
	"net/http"
	log "github.com/sirupsen/logrus"
)


var crc_table = []byte{
0, 7, 14, 9, 28, 27, 18, 21,
56, 63, 54, 49, 36, 35, 42, 45,
112, 119, 126, 121, 108, 107, 98, 101,
72, 79, 70, 65, 84, 83, 90, 93,
224, 231, 238, 233, 252, 251, 242, 245,
216, 223, 214, 209, 196, 195, 202, 205,
144, 151, 158, 153, 140, 139, 130, 133,
168, 175, 166, 161, 180, 179, 186, 189,
199, 192, 201, 206, 219, 220, 213, 210,
255, 248, 241, 246, 227, 228, 237, 234,
183, 176, 185, 190, 171, 172, 165, 162,
143, 136, 129, 134, 147, 148, 157, 154,
39, 32, 41, 46, 59, 60, 53, 50,
31, 24, 17, 22, 3, 4, 13, 10,
87, 80, 89, 94, 75, 76, 69, 66,
111, 104, 97, 102, 115, 116, 125, 122,
137, 142, 135, 128, 149, 146, 155, 156,
177, 182, 191, 184, 173, 170, 163, 164,
249, 254, 247, 240, 229, 226, 235, 236,
193, 198, 207, 200, 221, 218, 211, 212,
105, 110, 103, 96, 117, 114, 123, 124,
81, 86, 95, 88, 77, 74, 67, 68,
25, 30, 23, 16, 5, 2, 11, 12,
33, 38, 47, 40, 61, 58, 51, 52,
78, 73, 64, 71, 82, 85, 92, 91,
118, 113, 120, 127, 106, 109, 100, 99,
62, 57, 48, 55, 34, 37, 44, 43,
6, 1, 8, 15, 26, 29, 20, 19,
174, 169, 160, 167, 178, 181, 188, 187,
150, 145, 152, 159, 138, 141, 132, 131,
222, 217, 208, 215, 194, 197, 204, 203,
230, 225, 232, 239, 250, 253, 244, 243,
}

func fw_crc(NewData byte, buf byte) (byte){
	buf = crc_table[(buf ^ NewData)]
	return buf
}

func cmd_write_single(addr byte, cmd byte, tx byte) ([2]byte){
	var txbuf [2]byte
	waddr :=  addr << 1
	var pec byte
	pec = 0
	pec = fw_crc(waddr, pec)
	pec = fw_crc(cmd, pec)
	pec = fw_crc(tx, pec)

	txbuf[0] = tx
	txbuf[1] = pec

	return txbuf
}

func to_twoscomplement(bits uint16, value int16) (int16){
	if value < 0 {
		value = (1<<bits) + value
	}
    return value
}

func voutmode_convert(exponent_count_20h byte, Yhbyte_8bh byte, Ylbyte_8bh byte) (float64){
    var a int16 = 0
	if (exponent_count_20h&(1<<4) != 0){
		a = - to_twoscomplement(5, int16(exponent_count_20h<<3>>3) * -1)
	} else {
		a = to_twoscomplement(5, int16(exponent_count_20h<<3>>3))
	}
    Y := binary.LittleEndian.Uint16([]byte{Ylbyte_8bh, Yhbyte_8bh})
    return float64(Y) * math.Pow(2, float64(a))
}

func linear_format(Yhbyte_8bh byte, Ylbyte_8bh byte) (float64){
	dataSum := binary.LittleEndian.Uint16([]byte{Ylbyte_8bh, Yhbyte_8bh})
    var x int16 = 0;
	if (dataSum&(1<<15) != 0) {
        x = -to_twoscomplement(5, int16(dataSum>>11) * -1)
	} else {
		x = to_twoscomplement(5, int16(dataSum>>11))
	}
	var y int16 = 0;

	if (dataSum&(1<<10) != 0) {
		y = -to_twoscomplement(5, int16(dataSum<<5>>5) * -1)
	} else {
		y = to_twoscomplement(5, int16(dataSum<<5>>5))
	}
	return float64(y) * math.Pow(2, float64(x))

}

type PowerData struct {
	Voltage float64
	Current float64
	Power float64
}

type PSUData struct {

	Status [2]byte

	FanSpeed float64
	Temperature1 float64
	Temperature2 float64

	Input PowerData

    Output PowerData
}

type BackBoardData struct {

	Temperature float64
	Status [2]byte

	Output12V1 PowerData
	Output12V2 PowerData
	Output12V3 PowerData
	Output5V PowerData
	Output33V PowerData

}

func collectPsuData(addr byte) PSUData {

	var psudata PSUData

	conn, err := smbus.Open(0, addr);
	if err != nil {
		fmt.Printf("open error: %v\n", err)
	}
	defer conn.Close()

	var status [2]byte;
	var ac_in [2]byte;
	var fan_speed [2]byte;
	var temp_1 [2]byte;
	var temp_2 [2]byte;
	var iout [2]byte;
	var iin [2]byte;
	var pout [2]byte;
	var pin [2]byte;
	var vout_exp [1]byte;
	var vout_mantisa [2]byte;

	conn.ReadBlockData(addr, 0x79, status[:]) // STATUS_WORD[2]
	conn.ReadBlockData(addr, 0x88, ac_in[:]) // READ_VIN[2]
	conn.ReadBlockData(addr, 0x90, fan_speed[:]) // READ_FAN_SPEED_1[2]
	conn.ReadBlockData(addr, 0x8d, temp_1[:]) // READ_TEMPARATURE_1[2]
	conn.ReadBlockData(addr, 0x8e, temp_2[:]) // READ_TEMPARATURE_2[2]
	conn.ReadBlockData(addr, 0x8c, iout[:]) // READ_IOUT[2]
	conn.ReadBlockData(addr, 0x89, iin[:]) // READ_IIN[2]
	conn.ReadBlockData(addr, 0x96, pout[:]) // READ_POUT[2]
	conn.ReadBlockData(addr, 0x97, pin[:]) // READ_PIN[2]
	conn.ReadBlockData(addr, 0x20, vout_exp[:]) // VOUT_MODE[1]
	conn.ReadBlockData(addr, 0x8B, vout_mantisa[:]) // READ_VOUT[2]


	psudata.Status = status
	psudata.Input.Voltage = linear_format(ac_in[1], ac_in[0])
	psudata.Input.Current = linear_format(iin[1], iin[0])
	// psudata.Input.Power = linear_format(iin[1], iin[0])
	psudata.Input.Power = psudata.Input.Voltage * psudata.Input.Current * 0.88;

	psudata.Output.Voltage = voutmode_convert(vout_exp[0], vout_mantisa[1], vout_mantisa[0])
	psudata.Output.Current = linear_format(iout[1], iout[0])
	psudata.Output.Power = linear_format(pout[1], pout[0])


	psudata.FanSpeed = linear_format(fan_speed[1], fan_speed[0])
	psudata.Temperature1 = linear_format(temp_1[1], temp_1[0])
	psudata.Temperature2 = linear_format(temp_2[1], temp_2[0])

	return psudata
}

func collectBackBoardData(addr byte) BackBoardData {
	var bbdata BackBoardData

	bb, err := smbus.Open(0, addr)
	if err != nil {
		fmt.Printf("open error: %v\n", err)
	}
	defer bb.Close()

	var bb_temp [2]byte;
	bb.ReadBlockData(addr, 0x8D, bb_temp[:])

	bbdata.Temperature = linear_format(bb_temp[1], bb_temp[0])

	var bb_status [2] byte;
	bb.ReadBlockData(addr, 0x79, bb_status[:])
	bbdata.Status = bb_status

	// Collect power rail data

	var tx [2]byte

	// 12V1 0x00 page
	tx = cmd_write_single(addr, 0x00, 0x00)
	bb.WriteBlockData(addr, 0x00, tx[:])

	var bb_v12_1_exp [1]byte;
	var bb_v12_1_out [2]byte;
	var bb_i12_1_out [3]byte;

	bb.ReadBlockData(addr, 0x20, bb_v12_1_exp[:])
	bb.ReadBlockData(addr, 0x8B, bb_v12_1_out[:])
	bb.ReadBlockData(addr, 0x8C, bb_i12_1_out[:])

	bbdata.Output12V1.Voltage = voutmode_convert(bb_v12_1_exp[0], bb_v12_1_out[1], bb_v12_1_out[0])
	bbdata.Output12V1.Current = linear_format(bb_i12_1_out[1], bb_i12_1_out[0])
	bbdata.Output12V1.Power = bbdata.Output12V1.Voltage * bbdata.Output12V1.Current

	// 12V2 0x01 page
	tx = cmd_write_single(addr, 0x00, 0x01)
	bb.WriteBlockData(addr, 0x00, tx[:])

	var bb_v12_2_exp [1]byte;
	var bb_v12_2_out [2]byte;
	var bb_i12_2_out [3]byte;

	bb.ReadBlockData(addr, 0x20, bb_v12_2_exp[:])
	bb.ReadBlockData(addr, 0x8B, bb_v12_2_out[:])
	bb.ReadBlockData(addr, 0x8C, bb_i12_2_out[:])

	bbdata.Output12V2.Voltage = voutmode_convert(bb_v12_2_exp[0], bb_v12_2_out[1], bb_v12_2_out[0])
	bbdata.Output12V2.Current = linear_format(bb_i12_2_out[1], bb_i12_2_out[0])
	bbdata.Output12V2.Power = bbdata.Output12V2.Voltage * bbdata.Output12V2.Current
	// 12V3 0x02 page
	tx = cmd_write_single(addr, 0x00, 0x02)
	bb.WriteBlockData(addr, 0x00, tx[:])

	var bb_v12_3_exp [1]byte;
	var bb_v12_3_out [2]byte;
	var bb_i12_3_out [3]byte;

	bb.ReadBlockData(addr, 0x20, bb_v12_3_exp[:])
	bb.ReadBlockData(addr, 0x8B, bb_v12_3_out[:])
	bb.ReadBlockData(addr, 0x8C, bb_i12_3_out[:])

	bbdata.Output12V3.Voltage = voutmode_convert(bb_v12_3_exp[0], bb_v12_3_out[1], bb_v12_3_out[0])
	bbdata.Output12V3.Current = linear_format(bb_i12_3_out[1], bb_i12_3_out[0])

	bbdata.Output12V3.Power = bbdata.Output12V3.Voltage * bbdata.Output12V3.Current

	// 5V1 0x10 page
	tx = cmd_write_single(addr, 0x00, 0x10)
	bb.WriteBlockData(addr, 0x00, tx[:])

	var bb_v5_exp [1]byte;
	var bb_v5_out [2]byte;
	var bb_i5_out [3]byte;

	bb.ReadBlockData(addr, 0x20, bb_v5_exp[:])
	bb.ReadBlockData(addr, 0x8B, bb_v5_out[:])
	bb.ReadBlockData(addr, 0x8C, bb_i5_out[:])

	bbdata.Output5V.Voltage = voutmode_convert(bb_v5_exp[0], bb_v5_out[1], bb_v5_out[0])
	bbdata.Output5V.Current = linear_format(bb_i5_out[1], bb_i5_out[0])
	bbdata.Output5V.Power = bbdata.Output5V.Voltage * bbdata.Output5V.Current

	// 33V1 0x10 page
	tx = cmd_write_single(addr, 0x00, 0x11)
	bb.WriteBlockData(addr, 0x00, tx[:])
	var bb_v33_exp [1]byte;
	var bb_v33_out [2]byte;
	var bb_i33_out [3]byte;

	bb.ReadBlockData(addr, 0x20, bb_v33_exp[:])
	bb.ReadBlockData(addr, 0x8B, bb_v33_out[:])
	bb.ReadBlockData(addr, 0x8C, bb_i33_out[:])

	bbdata.Output33V.Voltage = voutmode_convert(bb_v33_exp[0], bb_v33_out[1], bb_v33_out[0])
	bbdata.Output33V.Current = linear_format(bb_i33_out[1], bb_i33_out[0])
	bbdata.Output33V.Power = bbdata.Output33V.Voltage * bbdata.Output33V.Current

	return bbdata
}

func main(){

	psu1 := newPsu1Collector()
	prometheus.MustRegister(psu1)

	http.Handle("/metrics", promhttp.Handler())
	log.Info("Beginning to serve on port :9103")
	log.Fatal(http.ListenAndServe(":9103", nil))

}
