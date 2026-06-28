package main

// pctEncode — percent-encoding query-параметра (побайтно → корректно для UTF-8).
func pctEncode(s string) string {
	const hex = "0123456789ABCDEF"
	var b []byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') ||
			c == '-' || c == '_' || c == '.' || c == '~' {
			b = append(b, c)
		} else {
			b = append(b, '%', hex[c>>4], hex[c&0x0f])
		}
	}
	return string(b)
}

func i64toa(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func pad4(n int64) string {
	s := i64toa(n)
	for len(s) < 4 {
		s = "0" + s
	}
	return s
}

// ftoa — float в строку с 4 знаками после запятой (для координат).
func ftoa(f float64) string {
	neg := f < 0
	if neg {
		f = -f
	}
	ip := int64(f)
	fracI := int64((f-float64(ip))*10000 + 0.5)
	if fracI >= 10000 {
		ip++
		fracI -= 10000
	}
	s := i64toa(ip) + "." + pad4(fracI)
	if neg {
		s = "-" + s
	}
	return s
}

// wmoText — код WMO weather_code → русское описание.
func wmoText(code int) string {
	switch code {
	case 0:
		return "ясно"
	case 1:
		return "преимущественно ясно"
	case 2:
		return "переменная облачность"
	case 3:
		return "пасмурно"
	case 45, 48:
		return "туман"
	case 51, 53, 55:
		return "морось"
	case 56, 57:
		return "ледяная морось"
	case 61, 63, 65:
		return "дождь"
	case 66, 67:
		return "ледяной дождь"
	case 71, 73, 75:
		return "снег"
	case 77:
		return "снежная крупа"
	case 80, 81, 82:
		return "ливень"
	case 85, 86:
		return "снежный ливень"
	case 95:
		return "гроза"
	case 96, 99:
		return "гроза с градом"
	}
	return "неизвестно"
}
