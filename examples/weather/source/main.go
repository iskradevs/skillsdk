//go:build tinygo

package main

import (
	"encoding/json"
	"unsafe"
)

func main() {}

const userAgent = "iskra-skill-weather/1.0"

type okEnv struct {
	OK     bool            `json:"ok"`
	Result json.RawMessage `json:"result"`
}

type errEnv struct {
	OK    bool    `json:"ok"`
	Error errBody `json:"error"`
}

type errBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type currentResult struct {
	Location     string  `json:"location"`
	Country      string  `json:"country"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
	TemperatureC float64 `json:"temperature_c"`
	ApparentC    float64 `json:"apparent_c"`
	HumidityPct  float64 `json:"humidity_pct"`
	WindKmh      float64 `json:"wind_kmh"`
	Condition    string  `json:"condition"`
	ObservedAt   string  `json:"observed_at"`
}

type forecastDay struct {
	Date      string  `json:"date"`
	MinC      float64 `json:"min_c"`
	MaxC      float64 `json:"max_c"`
	Condition string  `json:"condition"`
}

type forecastResult struct {
	Location string        `json:"location"`
	Country  string        `json:"country"`
	Daily    []forecastDay `json:"daily"`
}

//go:wasmexport handle
func handle(argsPtr, argsLen int32) int64 {
	input := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(uint32(argsPtr)))), argsLen)
	var in struct {
		Tool string          `json:"tool"`
		Args json.RawMessage `json:"args"`
	}
	if err := json.Unmarshal(input, &in); err != nil {
		return errEnvelope("bad_input", "не удалось разобрать ввод")
	}
	switch in.Tool {
	case "weather_current":
		return weatherCurrent(in.Args)
	case "weather_forecast":
		return weatherForecast(in.Args)
	}
	return errEnvelope("unknown_tool", "неизвестный инструмент")
}

func weatherCurrent(args json.RawMessage) int64 {
	var a struct {
		Location string `json:"location"`
	}
	_ = json.Unmarshal(args, &a)
	if a.Location == "" {
		return errEnvelope("bad_input", "не указан город (location)")
	}
	lat, lon, place, country, ok := geocode(a.Location)
	if !ok {
		return errEnvelope("not_found", "город не найден")
	}
	body, ok := httpGet("https://api.open-meteo.com/v1/forecast?latitude=" + ftoa(lat) +
		"&longitude=" + ftoa(lon) +
		"&current=temperature_2m,relative_humidity_2m,apparent_temperature,weather_code,wind_speed_10m&timezone=auto")
	if !ok {
		return errEnvelope("upstream", "сервис погоды недоступен")
	}
	var f struct {
		Current struct {
			Time     string  `json:"time"`
			Temp     float64 `json:"temperature_2m"`
			Humidity float64 `json:"relative_humidity_2m"`
			Apparent float64 `json:"apparent_temperature"`
			Code     int     `json:"weather_code"`
			Wind     float64 `json:"wind_speed_10m"`
		} `json:"current"`
	}
	if json.Unmarshal([]byte(body), &f) != nil {
		return errEnvelope("upstream", "не удалось разобрать ответ погоды")
	}
	return okEnvelope(currentResult{
		Location: place, Country: country, Latitude: lat, Longitude: lon,
		TemperatureC: f.Current.Temp, ApparentC: f.Current.Apparent,
		HumidityPct: f.Current.Humidity, WindKmh: f.Current.Wind,
		Condition: wmoText(f.Current.Code), ObservedAt: f.Current.Time,
	})
}

func weatherForecast(args json.RawMessage) int64 {
	var a struct {
		Location string `json:"location"`
		Days     int    `json:"days"`
	}
	_ = json.Unmarshal(args, &a)
	if a.Location == "" {
		return errEnvelope("bad_input", "не указан город (location)")
	}
	days := a.Days
	if days < 1 {
		days = 3
	}
	if days > 7 {
		days = 7
	}
	lat, lon, place, country, ok := geocode(a.Location)
	if !ok {
		return errEnvelope("not_found", "город не найден")
	}
	body, ok := httpGet("https://api.open-meteo.com/v1/forecast?latitude=" + ftoa(lat) +
		"&longitude=" + ftoa(lon) +
		"&daily=weather_code,temperature_2m_max,temperature_2m_min&forecast_days=" + i64toa(int64(days)) + "&timezone=auto")
	if !ok {
		return errEnvelope("upstream", "сервис погоды недоступен")
	}
	var f struct {
		Daily struct {
			Time []string  `json:"time"`
			Code []int     `json:"weather_code"`
			Max  []float64 `json:"temperature_2m_max"`
			Min  []float64 `json:"temperature_2m_min"`
		} `json:"daily"`
	}
	if json.Unmarshal([]byte(body), &f) != nil {
		return errEnvelope("upstream", "не удалось разобрать ответ погоды")
	}
	out := make([]forecastDay, 0, len(f.Daily.Time))
	for i := range f.Daily.Time {
		if i >= len(f.Daily.Code) || i >= len(f.Daily.Max) || i >= len(f.Daily.Min) {
			break
		}
		out = append(out, forecastDay{
			Date: f.Daily.Time[i], MinC: f.Daily.Min[i], MaxC: f.Daily.Max[i],
			Condition: wmoText(f.Daily.Code[i]),
		})
	}
	return okEnvelope(forecastResult{Location: place, Country: country, Daily: out})
}

func geocode(name string) (lat, lon float64, place, country string, ok bool) {
	body, got := httpGet("https://geocoding-api.open-meteo.com/v1/search?name=" + pctEncode(name) + "&count=1&language=ru&format=json")
	if !got {
		return 0, 0, "", "", false
	}
	var g struct {
		Results []struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
			Name      string  `json:"name"`
			Country   string  `json:"country"`
		} `json:"results"`
	}
	if json.Unmarshal([]byte(body), &g) != nil || len(g.Results) == 0 {
		return 0, 0, "", "", false
	}
	r := g.Results[0]
	return r.Latitude, r.Longitude, r.Name, r.Country, true
}

func httpGet(url string) (string, bool) {
	resp, err := HTTP(HTTPRequest{Method: "GET", URL: url, Headers: map[string]string{"User-Agent": userAgent}})
	if err != nil || resp.Status != 200 {
		return "", false
	}
	return resp.Body, true
}

func okEnvelope(result any) int64 {
	rb, err := json.Marshal(result)
	if err != nil {
		return errEnvelope("marshal", "не удалось сериализовать результат")
	}
	b, _ := json.Marshal(okEnv{OK: true, Result: rb})
	return writeJSON(b)
}

func errEnvelope(code, message string) int64 {
	b, _ := json.Marshal(errEnv{OK: false, Error: errBody{Code: code, Message: message}})
	return writeJSON(b)
}
