package main

import "testing"

func TestDaysFromCivilRoundTrip(t *testing.T) {
	cases := [][3]int{{1970, 1, 1}, {2000, 2, 29}, {2026, 6, 15}, {1969, 12, 31}}
	for _, c := range cases {
		z := daysFromCivil(c[0], c[1], c[2])
		y, m, d := civilFromDays(z)
		if y != c[0] || m != c[1] || d != c[2] {
			t.Fatalf("roundtrip %v → %d-%02d-%02d", c, y, m, d)
		}
	}
}

func TestDayNumbers(t *testing.T) {
	if daysFromCivil(1970, 1, 1) != 0 {
		t.Fatalf("epoch != 0")
	}
	if daysFromCivil(2026, 6, 16)-daysFromCivil(2026, 6, 15) != 1 {
		t.Fatalf("diff of consecutive days != 1")
	}
}

func TestWeekday(t *testing.T) {
	if w := weekday(daysFromCivil(2026, 6, 15)); w != 1 { // понедельник
		t.Fatalf("2026-06-15 weekday=%d, ожидался 1 (пн)", w)
	}
	if w := weekday(daysFromCivil(2026, 6, 14)); w != 0 { // воскресенье
		t.Fatalf("2026-06-14 weekday=%d, ожидался 0 (вс)", w)
	}
}

func TestParseFmtDate(t *testing.T) {
	y, m, d, ok := parseDate("2026-06-15")
	if !ok || y != 2026 || m != 6 || d != 15 {
		t.Fatalf("parseDate: %d %d %d %v", y, m, d, ok)
	}
	if _, _, _, ok := parseDate("2026-13-01"); ok {
		t.Fatalf("parseDate должен отвергнуть месяц 13")
	}
	if _, _, _, ok := parseDate("bad"); ok {
		t.Fatalf("parseDate должен отвергнуть мусор")
	}
	if fmtDate(2026, 6, 5) != "2026-06-05" {
		t.Fatalf("fmtDate")
	}
	if fmtCompact(2026, 6, 5) != "20260605" {
		t.Fatalf("fmtCompact")
	}
}

func TestValidDayoffBody(t *testing.T) {
	if !validDayoffBody("0110", 4) {
		t.Fatalf("валидное тело из 4 статусов должно приниматься")
	}
	if !validDayoffBody("024", 3) { // рабочий, сокращённый, covid-рабочий
		t.Fatalf("статусы 0/2/4 валидны")
	}
	if validDayoffBody("100", 10) { // код ошибки: длина != calendar
		t.Fatalf("несовпадение длины должно отвергаться")
	}
	if validDayoffBody("199", 3) { // код ошибки с посторонним '9'
		t.Fatalf("байт вне {0,1,2,4} должен отвергаться")
	}
	if validDayoffBody("", 3) {
		t.Fatalf("пустое тело при ненулевом диапазоне должно отвергаться")
	}
}
