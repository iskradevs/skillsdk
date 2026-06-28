package main

// Алгоритмы civil-date по Говарду Хиннанту (days_from_civil/civil_from_days).
// Без пакета time: TinyGo wasm-unknown не даёт wall-clock. «Сегодня» — из Now().

func floorDiv(a, b int) int {
	q := a / b
	if (a%b != 0) && ((a < 0) != (b < 0)) {
		q--
	}
	return q
}

func daysFromCivil(y, m, d int) int {
	yy := y
	if m <= 2 {
		yy--
	}
	era := floorDiv(yy, 400)
	yoe := yy - era*400
	var mp int
	if m > 2 {
		mp = m - 3
	} else {
		mp = m + 9
	}
	doy := (153*mp+2)/5 + d - 1
	doe := yoe*365 + yoe/4 - yoe/100 + doy
	return era*146097 + doe - 719468
}

func civilFromDays(z int) (int, int, int) {
	z += 719468
	era := floorDiv(z, 146097)
	doe := z - era*146097
	yoe := (doe - doe/1460 + doe/36524 - doe/146096) / 365
	y := yoe + era*400
	doy := doe - (365*yoe + yoe/4 - yoe/100)
	mp := (5*doy + 2) / 153
	d := doy - (153*mp+2)/5 + 1
	var m int
	if mp < 10 {
		m = mp + 3
	} else {
		m = mp - 9
	}
	if m <= 2 {
		y++
	}
	return y, m, d
}

// weekday → 0=воскресенье .. 6=суббота.
func weekday(z int) int {
	return (((z + 4) % 7) + 7) % 7
}

var weekdayNames = []string{"воскресенье", "понедельник", "вторник", "среда", "четверг", "пятница", "суббота"}

func atoiN(s string) (int, bool) {
	if len(s) == 0 {
		return 0, false
	}
	n := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			return 0, false
		}
		n = n*10 + int(c-'0')
	}
	return n, true
}

func parseDate(s string) (int, int, int, bool) {
	if len(s) != 10 || s[4] != '-' || s[7] != '-' {
		return 0, 0, 0, false
	}
	y, ok1 := atoiN(s[0:4])
	m, ok2 := atoiN(s[5:7])
	d, ok3 := atoiN(s[8:10])
	if !ok1 || !ok2 || !ok3 || m < 1 || m > 12 || d < 1 || d > 31 {
		return 0, 0, 0, false
	}
	return y, m, d, true
}

func itoa2(n int) string {
	return string([]byte{byte('0' + (n/10)%10), byte('0' + n%10)})
}

func itoa4(n int) string {
	return string([]byte{
		byte('0' + (n/1000)%10), byte('0' + (n/100)%10),
		byte('0' + (n/10)%10), byte('0' + n%10),
	})
}

func fmtDate(y, m, d int) string    { return itoa4(y) + "-" + itoa2(m) + "-" + itoa2(d) }
func fmtCompact(y, m, d int) string { return itoa4(y) + itoa2(m) + itoa2(d) }

// validDayoffBody проверяет, что тело ответа isdayoff — ровно n символов статусов
// дней: '0'/'4' рабочий, '1' нерабочий, '2' сокращённый. Так отсекаются коды-ошибки
// API ("100"/"101"/"199") при несовпадении длины (n != 3) и любые посторонние байты
// (например '9' в "199"). Остаточная неоднозначность "100"/"101" с реальным
// 3-дневным диапазоном неразрешима по содержимому, но parseDate валидирует даты до
// запроса, поэтому такие коды-ошибки практически не возникают.
func validDayoffBody(body string, n int) bool {
	if len(body) != n {
		return false
	}
	for i := 0; i < len(body); i++ {
		switch body[i] {
		case '0', '1', '2', '4':
		default:
			return false
		}
	}
	return true
}
