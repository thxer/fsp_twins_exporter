package main

import (
	"encoding/binary"
	"github.com/prometheus/client_golang/prometheus"
)

//Define a struct for you collector that contains pointers
//to prometheus descriptors for each metric you wish to expose.
//Note you can also include fields of other types if they provide utility
//but we just won't be exposing them as metrics.
type psu1Collector struct {
	voltage *prometheus.Desc
	current *prometheus.Desc
	power *prometheus.Desc
	fanSpeed *prometheus.Desc
	temperature *prometheus.Desc
	psuStatus *prometheus.Desc

	bbStatus *prometheus.Desc
	bbTemperature *prometheus.Desc

	bbVoltage *prometheus.Desc
	bbCurrent *prometheus.Desc
	bbPower *prometheus.Desc
}

//You must create a constructor for you collector that
//initializes every descriptor and returns a pointer to the collector
func newPsu1Collector() *psu1Collector {
	return &psu1Collector{
		voltage: prometheus.NewDesc("psu_voltage",
			"PSU input voltage",
			[]string{"psu", "direction"}, nil,
		),
		current: prometheus.NewDesc("psu_current",
			"PSU input current",
			[]string{"psu", "direction"}, nil,
		),
		power: prometheus.NewDesc("psu_power",
			"PSU input current",
			[]string{"psu", "direction"}, nil,
		),
		fanSpeed: prometheus.NewDesc("psu_fan_speed",
			"PSU fan speed current",
			[]string{"psu"}, nil,
		),
		temperature: prometheus.NewDesc("psu_temperature",
			"PSU in temperature",
			[]string{"psu", "flow"}, nil,
		),
		psuStatus: prometheus.NewDesc("psu_status",
			"PSU status",
			[]string{"psu"}, nil,
		),
		bbStatus: prometheus.NewDesc("bb_status",
			"PSU status",
			nil, nil,
		),
		bbTemperature: prometheus.NewDesc("bb_temperature",
			"BB temperature",
			nil, nil,
		),
		bbVoltage: prometheus.NewDesc("bb_voltage",
			"BB voltage",
			[]string{"sensor"}, nil,
		),
		bbCurrent: prometheus.NewDesc("bb_current",
			"BB current",
			[]string{"sensor"}, nil,
		),
		bbPower: prometheus.NewDesc("bb_power",
			"BB power",
			[]string{"sensor"}, nil,
		),
	}
}

//Each and every collector must implement the Describe function.
//It essentially writes all descriptors to the prometheus desc channel.
func (collector *psu1Collector) Describe(ch chan<- *prometheus.Desc) {

	//Update this section with the each metric you create for a given collector
	ch <- collector.voltage
	ch <- collector.current
	ch <- collector.power
	ch <- collector.fanSpeed
	ch <- collector.temperature
	ch <- collector.psuStatus
	ch <- collector.bbStatus
	ch <- collector.bbTemperature
}

//Collect implements required collect function for all promehteus collectors
func (collector *psu1Collector) Collect(ch chan<- prometheus.Metric) {

	//Implement logic here to determine proper metric value to return to prometheus
	//for each descriptor or call other functions that do so.

	psu1data := collectPsuData(0x58)

	//Write latest value for each metric in the prometheus metric channel.
	//Note that you can pass CounterValue, GaugeValue, or UntypedValue types here.
	ch <- prometheus.MustNewConstMetric(collector.voltage, prometheus.GaugeValue, psu1data.Input.Voltage, "1", "input")
	ch <- prometheus.MustNewConstMetric(collector.current, prometheus.GaugeValue, psu1data.Input.Current,"1", "input")
	ch <- prometheus.MustNewConstMetric(collector.power, prometheus.GaugeValue, psu1data.Input.Power,"1", "input")
	ch <- prometheus.MustNewConstMetric(collector.voltage, prometheus.GaugeValue, psu1data.Output.Voltage,"1", "output")
	ch <- prometheus.MustNewConstMetric(collector.current, prometheus.GaugeValue, psu1data.Output.Current,"1", "output")
	ch <- prometheus.MustNewConstMetric(collector.power, prometheus.GaugeValue, psu1data.Output.Power,"1", "output")
	ch <- prometheus.MustNewConstMetric(collector.fanSpeed, prometheus.GaugeValue, psu1data.FanSpeed,"1")
	ch <- prometheus.MustNewConstMetric(collector.temperature, prometheus.GaugeValue, psu1data.Temperature1,"1", "in")
	ch <- prometheus.MustNewConstMetric(collector.temperature, prometheus.GaugeValue, psu1data.Temperature2,"1", "out")
	ch <- prometheus.MustNewConstMetric(collector.psuStatus, prometheus.GaugeValue, float64(binary.BigEndian.Uint16(psu1data.Status[:])),"1")

	psu2data := collectPsuData(0x59)

	//Write latest value for each metric in the prometheus metric channel.
	//Note that you can pass CounterValue, GaugeValue, or UntypedValue types here.
	ch <- prometheus.MustNewConstMetric(collector.voltage, prometheus.GaugeValue, psu2data.Input.Voltage, "2", "input")
	ch <- prometheus.MustNewConstMetric(collector.current, prometheus.GaugeValue, psu2data.Input.Current,"2", "input")
	ch <- prometheus.MustNewConstMetric(collector.power, prometheus.GaugeValue, psu2data.Input.Power,"2", "input")
	ch <- prometheus.MustNewConstMetric(collector.voltage, prometheus.GaugeValue, psu2data.Output.Voltage,"2", "output")
	ch <- prometheus.MustNewConstMetric(collector.current, prometheus.GaugeValue, psu2data.Output.Current,"2", "output")
	ch <- prometheus.MustNewConstMetric(collector.power, prometheus.GaugeValue, psu2data.Output.Power,"2", "output")
	ch <- prometheus.MustNewConstMetric(collector.fanSpeed, prometheus.GaugeValue, psu2data.FanSpeed,"2")
	ch <- prometheus.MustNewConstMetric(collector.temperature, prometheus.GaugeValue, psu2data.Temperature1,"2", "in")
	ch <- prometheus.MustNewConstMetric(collector.temperature, prometheus.GaugeValue, psu2data.Temperature1,"2", "out")
	ch <- prometheus.MustNewConstMetric(collector.psuStatus, prometheus.GaugeValue, float64(binary.BigEndian.Uint16(psu2data.Status[:])),"2")

	bbData := collectBackBoardData(0x25)

	ch <- prometheus.MustNewConstMetric(collector.bbTemperature, prometheus.GaugeValue, bbData.Temperature)
	ch <- prometheus.MustNewConstMetric(collector.bbStatus, prometheus.GaugeValue, float64(binary.BigEndian.Uint16(bbData.Status[:])))
	ch <- prometheus.MustNewConstMetric(collector.bbVoltage, prometheus.GaugeValue, bbData.Output12V1.Voltage, "12V1")
	ch <- prometheus.MustNewConstMetric(collector.bbVoltage, prometheus.GaugeValue, bbData.Output12V2.Voltage, "12V2")
	ch <- prometheus.MustNewConstMetric(collector.bbVoltage, prometheus.GaugeValue, bbData.Output12V3.Voltage, "12V3")
	ch <- prometheus.MustNewConstMetric(collector.bbVoltage, prometheus.GaugeValue, bbData.Output5V.Voltage, "5V")
	ch <- prometheus.MustNewConstMetric(collector.bbVoltage, prometheus.GaugeValue, bbData.Output33V.Voltage, "3.3V")
	ch <- prometheus.MustNewConstMetric(collector.bbCurrent, prometheus.GaugeValue, bbData.Output12V1.Current, "12V1")
	ch <- prometheus.MustNewConstMetric(collector.bbCurrent, prometheus.GaugeValue, bbData.Output12V2.Current, "12V2")
	ch <- prometheus.MustNewConstMetric(collector.bbCurrent, prometheus.GaugeValue, bbData.Output12V3.Current, "12V3")
	ch <- prometheus.MustNewConstMetric(collector.bbCurrent, prometheus.GaugeValue, bbData.Output5V.Current, "5V")
	ch <- prometheus.MustNewConstMetric(collector.bbCurrent, prometheus.GaugeValue, bbData.Output33V.Current, "3.3V")
	ch <- prometheus.MustNewConstMetric(collector.bbPower, prometheus.GaugeValue, bbData.Output12V1.Power, "12V1")
	ch <- prometheus.MustNewConstMetric(collector.bbPower, prometheus.GaugeValue, bbData.Output12V2.Power, "12V2")
	ch <- prometheus.MustNewConstMetric(collector.bbPower, prometheus.GaugeValue, bbData.Output12V3.Power, "12V3")
	ch <- prometheus.MustNewConstMetric(collector.bbPower, prometheus.GaugeValue, bbData.Output5V.Power, "5V")
	ch <- prometheus.MustNewConstMetric(collector.bbPower, prometheus.GaugeValue, bbData.Output33V.Power, "3.3V")

}
