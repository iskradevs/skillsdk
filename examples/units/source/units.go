package main

// Таблицы коэффициентов: единица → сколько базовых единиц в одной этой.
var unitTables = map[string]map[string]float64{
	"length": {"mm": 0.001, "cm": 0.01, "m": 1, "km": 1000, "in": 0.0254, "ft": 0.3048, "yd": 0.9144, "mi": 1609.344, "nmi": 1852},
	"mass":   {"mg": 0.001, "g": 1, "kg": 1000, "t": 1e6, "oz": 28.349523125, "lb": 453.59237},
	"area":   {"cm2": 0.0001, "m2": 1, "km2": 1e6, "ha": 1e4, "ft2": 0.09290304, "ac": 4046.8564224},
	"volume": {"ml": 0.001, "l": 1, "m3": 1000, "gal": 3.785411784, "qt": 0.946352946, "pt": 0.473176473, "cup": 0.2365882365},
	"speed":  {"m/s": 1, "km/h": 0.277777778, "mph": 0.44704, "kn": 0.514444444},
	"data":   {"B": 1, "KB": 1e3, "MB": 1e6, "GB": 1e9, "TB": 1e12, "KiB": 1024, "MiB": 1048576, "GiB": 1073741824},
	"time":   {"s": 1, "min": 60, "h": 3600, "d": 86400, "wk": 604800},
}

// convertUnits возвращает (результат, категория, ok). ok=false при неизвестных
// или несовместимых единицах.
func convertUnits(value float64, from, to string) (float64, string, bool) {
	if r, ok := convertTemp(value, from, to); ok {
		return r, "temperature", true
	}
	for cat, table := range unitTables {
		ff, ok1 := table[from]
		tt, ok2 := table[to]
		if ok1 && ok2 {
			return value * ff / tt, cat, true
		}
	}
	return 0, "", false
}

func convertTemp(v float64, from, to string) (float64, bool) {
	var c float64
	switch from {
	case "c", "C":
		c = v
	case "f", "F":
		c = (v - 32) * 5 / 9
	case "k", "K":
		c = v - 273.15
	default:
		return 0, false
	}
	switch to {
	case "c", "C":
		return c, true
	case "f", "F":
		return c*9/5 + 32, true
	case "k", "K":
		return c + 273.15, true
	}
	return 0, false
}
