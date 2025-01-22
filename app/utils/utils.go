package utils

import (
	"database/sql"
	"log"
	"math"
	"strconv"
)

// Prepara os dados para o template
func PrepareTemplateData(currentData, previousData map[string]interface{}) map[string]interface{} {
	if currentData == nil {
		return map[string]interface{}{
			"Message":           "Nenhum dado disponível no momento.",
			"WindDirection":     "N/D",
			"WindIconClass":     "rotate-0",
			"UVStatus":          "N/A",
			"HumidityStatus":    "N/A",
			"RainStatus":        "N/A",
			"TemperatureStatus": "N/A",
			"Temperature":       0.0,
			"UVIndex":           0.0,
			"Humidity":          0.0,
			"RainLevel":         0.0,
			"AverageWindSpeed":  0.0,
			"WindSpeedStatus":   "N/A",
			"WindSpeedKMH":      0.0,
		}
	}

	// Extrair os valores de currentData
	windDirectionRad := getFloatFromMap(currentData, "wind_direction")
	uvIndex := getFloatFromMap(currentData, "uv_index")
	humidity := getFloatFromMap(currentData, "humidity")
	currentRainLevel := getFloatFromMap(currentData, "rain_level")
	temperature := getFloatFromMap(currentData, "temperature")
	averageWindSpeed := getFloatFromMap(currentData, "average_wind_speed")

	// Calcular o nível de chuva anterior
	previousRainLevel := currentRainLevel
	if previousData != nil {
		previousRainLevel = getFloatFromMap(previousData, "rain_level")
	}

	// Converter radianos para direção e ícone
	windDirection, windIconClass := RadToDirectionWithIcon(windDirectionRad)

	// Preparar o contexto para o template
	return map[string]interface{}{
		"WindDirection":     windDirection,
		"WindIconClass":     windIconClass,
		"UVStatus":          GetUVStatus(uvIndex),
		"HumidityStatus":    GetHumidityStatus(humidity),
		"RainStatus":        GetRainStatus(currentRainLevel, previousRainLevel),
		"TemperatureStatus": GetTemperatureStatus(temperature),
		"Temperature":       temperature,
		"UVIndex":           uvIndex,
		"Humidity":          humidity,
		"RainLevel":         currentRainLevel,
		"AverageWindSpeed":  averageWindSpeed * 3.6, // Converter m/s para km/h
		"WindSpeedStatus":   GetWindSpeedStatus(averageWindSpeed * 3.6),
	}
}

func PrepareAPIData(currentData, previousData map[string]interface{}) map[string]interface{} {
	if currentData == nil {
		return map[string]interface{}{
			"temperature":        0.0,
			"temperature_status": "N/A",
			"humidity":           0.0,
			"humidity_status":    "N/A",
			"rain_level":         0.0,
			"rain_status":        "N/A",
			"uv_index":           0.0,
			"uv_status":          "N/A",
			"wind_speed_kmh":     0.0,
			"wind_speed_status":  "N/A",
			"wind_direction":     "N/D",
		}
	}

	windDirectionRad := getFloatFromMap(currentData, "wind_direction")
	windDirection, _ := RadToDirectionWithIcon(windDirectionRad)

	return map[string]interface{}{
		"temperature":        getFloatFromMap(currentData, "temperature"),
		"temperature_status": GetTemperatureStatus(getFloatFromMap(currentData, "temperature")),
		"humidity":           getFloatFromMap(currentData, "humidity"),
		"humidity_status":    GetHumidityStatus(getFloatFromMap(currentData, "humidity")),
		"rain_level":         getFloatFromMap(currentData, "rain_level"),
		"rain_status":        GetRainStatus(getFloatFromMap(currentData, "rain_level"), getFloatFromMap(previousData, "rain_level")),
		"uv_index":           getFloatFromMap(currentData, "uv_index"),
		"uv_status":          GetUVStatus(getFloatFromMap(currentData, "uv_index")),
		"wind_speed_kmh":     getFloatFromMap(currentData, "average_wind_speed") * 3.6,
		"wind_speed_status":  GetWindSpeedStatus(getFloatFromMap(currentData, "average_wind_speed") * 3.6),
		"wind_direction":     windDirection,
	}
}

// Função auxiliar para extrair valores float de um mapa
func getFloatFromMap(data map[string]interface{}, key string) float64 {
	if value, exists := data[key]; exists {
		switch v := value.(type) {
		case float64:
			return v
		case string:
			if parsed, err := strconv.ParseFloat(v, 64); err == nil {
				return parsed
			}
		}
	}
	return 0.0
}

// GetMySQLData retorna os dois últimos registros do banco
func GetMySQLData(db *sql.DB) (map[string]interface{}, map[string]interface{}) {
	rows, err := db.Query(`
        SELECT 
            rain_level, 
            average_wind_speed,  -- Nome correto da coluna
            wind_direction,
            humidity,
            uv_index,
            temperature,
            timestamp
        FROM sensor_data
        ORDER BY timestamp DESC
        LIMIT 2
    `)
	if err != nil {
		log.Printf("Erro na query: %v", err) // 👈 Log de erro
		return nil, nil
	}
	defer rows.Close()

	var results []map[string]interface{}

	for rows.Next() {
		// Use tipos específicos para evitar NULL
		var (
			rainLevel        sql.NullFloat64
			averageWindSpeed sql.NullFloat64
			windDirection    sql.NullFloat64
			humidity         sql.NullFloat64
			uvIndex          sql.NullFloat64
			temperature      sql.NullFloat64
			timestamp        sql.NullInt64
		)

		// Scan com tipos seguros
		if err := rows.Scan(
			&rainLevel,
			&averageWindSpeed,
			&windDirection,
			&humidity,
			&uvIndex,
			&temperature,
			&timestamp,
		); err != nil {
			log.Printf("Erro no scan: %v", err) // 👈 Log de erro
			return nil, nil
		}

		// Mapeamento seguro
		data := map[string]interface{}{
			"rain_level":         rainLevel.Float64,
			"average_wind_speed": averageWindSpeed.Float64,
			"wind_direction":     windDirection.Float64,
			"humidity":           humidity.Float64,
			"uv_index":           uvIndex.Float64,
			"temperature":        temperature.Float64,
			"timestamp":          timestamp.Int64,
		}

		log.Printf("Dado processado: %+v", data) // 👈 Log detalhado
		results = append(results, data)
	}

	if len(results) >= 2 {
		return results[0], results[1]
	} else if len(results) == 1 {
		return results[0], nil
	}

	log.Println("Nenhum dado encontrado") // 👈 Log informativo
	return nil, nil
}

// RadToDirectionWithIcon converte radianos para direção cardeal
func RadToDirectionWithIcon(rad float64) (string, string) {
	directions := []struct {
		Name string
		Icon string
	}{
		{"Norte", "rotate-0"},
		{"Nordeste", "rotate-45"},
		{"Leste", "rotate-90"},
		{"Sudeste", "rotate-135"},
		{"Sul", "rotate-180"},
		{"Sudoeste", "rotate-225"},
		{"Oeste", "rotate-270"},
		{"Noroeste", "rotate-315"},
	}
	rad = math.Mod(rad, 2*math.Pi)
	index := (int((rad+math.Pi/8)/(math.Pi/4))%8 + 8) % 8
	return directions[index].Name, directions[index].Icon
}

// Cálculo de média com validação de array vazio
func CalculateAverage(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}
	sum := 0.0
	for _, value := range values {
		sum += value
	}
	return sum / float64(len(values))
}

func CalculateMax(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}
	max := values[0]
	for _, value := range values {
		if value > max {
			max = value
		}
	}
	return max
}

func CalculateMin(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}
	min := values[0]
	for _, value := range values {
		if value < min {
			min = value
		}
	}
	return min
}

// GetUVStatus determina o status da radiação UV
func GetUVStatus(uvIndex float64) string {
	switch {
	case uvIndex < 3:
		return "Níveis baixos de UV"
	case uvIndex < 6:
		return "Níveis moderados de UV"
	case uvIndex < 8:
		return "Níveis altos de UV"
	case uvIndex < 11:
		return "Níveis muito altos de UV"
	default:
		return "Risco extremo de UV"
	}
}

func GetHumidityStatus(humidity float64) string {
	switch {
	case humidity < 30:
		return "Ar muito seco"
	case humidity < 60:
		return "Umidade confortável"
	default:
		return "Ar muito úmido"
	}
}

func GetRainStatus(currentRainLevel, previousRainLevel float64) string {
	rainDifference := currentRainLevel - previousRainLevel
	switch {
	case rainDifference <= 0:
		return "Não está chovendo"
	case rainDifference < 0.0006:
		return "Chuviscando"
	default:
		return "Chovendo"
	}
}

func GetTemperatureStatus(temperature float64) string {
	switch {
	case temperature < 10:
		return "Frio intenso"
	case temperature < 20:
		return "Clima frio"
	case temperature < 25:
		return "Clima agradável"
	case temperature < 30:
		return "Clima quente"
	default:
		return "Calor extremo"
	}
}

func GetWindSpeedStatus(windSpeedKMH float64) string {
	switch {
	case windSpeedKMH == 0:
		return "Sem vento"
	case windSpeedKMH < 12.0:
		return "Brisa leve"
	case windSpeedKMH < 20.0:
		return "Vento fresco"
	case windSpeedKMH < 41.0:
		return "Vento moderado"
	case windSpeedKMH < 62.0:
		return "Vento forte"
	case windSpeedKMH < 75.0:
		return "Vento muito forte"
	case windSpeedKMH < 103.0:
		return "Vendaval severo"
	case windSpeedKMH < 120.0:
		return "Tempestade"
	default:
		return "Ciclone tropical"
	}
}
