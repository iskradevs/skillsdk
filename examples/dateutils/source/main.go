//go:build tinygo

package main

import (
	"encoding/json"
	"unsafe"
)

func main() {}

const userAgent = "iskra-skill-dateutils/1.0"

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

type diffResult struct {
	From string `json:"from"`
	To   string `json:"to"`
	Days int    `json:"days"`
}

type addResult struct {
	Date    string `json:"date"`
	Days    int    `json:"days"`
	Result  string `json:"result"`
	Weekday string `json:"weekday"`
}

type workingResult struct {
	From         string `json:"from"`
	To           string `json:"to"`
	CalendarDays int    `json:"calendar_days"`
	WorkingDays  int    `json:"working_days"`
	Weekends     int    `json:"weekends"`
	Holidays     int    `json:"holidays"`
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
	case "date_diff":
		return dateDiff(in.Args)
	case "date_add":
		return dateAdd(in.Args)
	case "working_days":
		return workingDays(in.Args)
	}
	return errEnvelope("unknown_tool", "неизвестный инструмент")
}

func dateDiff(args json.RawMessage) int64 {
	var a struct {
		From string `json:"from"`
		To   string `json:"to"`
	}
	_ = json.Unmarshal(args, &a)
	fy, fm, fd, ok1 := parseDate(a.From)
	ty, tm, td, ok2 := parseDate(a.To)
	if !ok1 || !ok2 {
		return errEnvelope("bad_input", "даты в формате YYYY-MM-DD")
	}
	days := daysFromCivil(ty, tm, td) - daysFromCivil(fy, fm, fd)
	return okEnvelope(diffResult{From: a.From, To: a.To, Days: days})
}

func dateAdd(args json.RawMessage) int64 {
	var a struct {
		Date string `json:"date"`
		Days int    `json:"days"`
	}
	_ = json.Unmarshal(args, &a)
	base := a.Date
	if base == "" {
		y, m, d := today()
		base = fmtDate(y, m, d)
	}
	y, m, d, ok := parseDate(base)
	if !ok {
		return errEnvelope("bad_input", "дата в формате YYYY-MM-DD")
	}
	z := daysFromCivil(y, m, d) + a.Days
	ry, rm, rd := civilFromDays(z)
	return okEnvelope(addResult{Date: base, Days: a.Days, Result: fmtDate(ry, rm, rd), Weekday: weekdayNames[weekday(z)]})
}

func workingDays(args json.RawMessage) int64 {
	var a struct {
		From string `json:"from"`
		To   string `json:"to"`
	}
	_ = json.Unmarshal(args, &a)
	fy, fm, fd, ok1 := parseDate(a.From)
	ty, tm, td, ok2 := parseDate(a.To)
	if !ok1 || !ok2 {
		return errEnvelope("bad_input", "даты в формате YYYY-MM-DD")
	}
	z0 := daysFromCivil(fy, fm, fd)
	z1 := daysFromCivil(ty, tm, td)
	if z1 < z0 {
		return errEnvelope("bad_input", "to раньше from")
	}
	calendar := z1 - z0 + 1
	url := "https://isdayoff.ru/api/getdata?date1=" + fmtCompact(fy, fm, fd) + "&date2=" + fmtCompact(ty, tm, td)
	body, ok := httpGet(url)
	if !ok || !validDayoffBody(body, calendar) {
		return errEnvelope("upstream", "производственный календарь недоступен")
	}
	working, nonWorking, weekends := 0, 0, 0
	for i := 0; i < calendar; i++ {
		if w := weekday(z0 + i); w == 0 || w == 6 {
			weekends++
		}
		switch body[i] {
		case '0', '2', '4':
			working++
		case '1':
			nonWorking++
		}
	}
	holidays := nonWorking - weekends
	if holidays < 0 {
		holidays = 0
	}
	return okEnvelope(workingResult{
		From: a.From, To: a.To, CalendarDays: calendar,
		WorkingDays: working, Weekends: weekends, Holidays: holidays,
	})
}

func today() (int, int, int) {
	ns, err := Now()
	if err != nil {
		return 1970, 1, 1
	}
	return civilFromDays(int(ns / 86400000000000))
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
