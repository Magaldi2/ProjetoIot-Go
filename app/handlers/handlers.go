package handlers

import (
	"database/sql"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
	"projeto/app/utils"
	"time"
)

// DatabaseConfig retorna a configuração de conexão com o banco
func DatabaseConfig() string {
	host := os.Getenv("MYSQL_HOST")
	user := os.Getenv("MYSQL_USER")
	password := os.Getenv("MYSQL_PASSWORD")
	database := os.Getenv("MYSQL_DB")
	return user + ":" + password + "@tcp(" + host + ":3306)/" + database
}

// Index Handler para a rota principal
func Index(templates *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		templates.ExecuteTemplate(w, "index.html", nil)
	}
}

func ApiIndexHandler(w http.ResponseWriter, r *http.Request) {
	db, err := sql.Open("mysql", DatabaseConfig())
	if err != nil {
		respondWithError(w, "Erro ao conectar ao banco", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	currentData, previousData := utils.GetMySQLData(db)
	context := utils.PrepareAPIData(currentData, previousData)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(context); err != nil {
		respondWithError(w, "Erro ao serializar dados", http.StatusInternalServerError)
	}
}

func Dashboard(templates *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		db, err := sql.Open("mysql", DatabaseConfig())
		if err != nil {
			log.Printf("Erro ao conectar ao banco: %v", err)
			http.Error(w, "Erro interno", http.StatusInternalServerError)
			return
		}
		defer db.Close()

		// Cálculo do período (UTC-3)
		now := time.Now().UTC()
		localNow := now.Add(-3 * time.Hour)
		startDay := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, time.UTC)
		startTimestamp := startDay.Unix() + 3*3600
		endTimestamp := startTimestamp + 24*3600

		// Buscar dados
		query := `
            SELECT timestamp, temperature, humidity, rain_level, average_wind_speed 
            FROM sensor_data 
            WHERE timestamp BETWEEN ? AND ? 
            ORDER BY timestamp
        `
		rows, err := db.Query(query, startTimestamp, endTimestamp)
		if err != nil {
			log.Printf("Erro ao buscar dados: %v", err)
			http.Error(w, "Erro interno", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		// Estrutura para armazenar os dados
		type SensorData struct {
			Timestamps  []string  `json:"timestamps"`
			Temperature []float64 `json:"temperature"`
			Humidity    []float64 `json:"humidity"`
			RainLevel   []float64 `json:"rain_level"`
			WindSpeed   []float64 `json:"wind_speed"`
		}

		var sensorData SensorData

		for rows.Next() {
			var timestamp int64
			var temp, hum, rain, wind sql.NullFloat64

			if err := rows.Scan(&timestamp, &temp, &hum, &rain, &wind); err != nil {
				log.Printf("Erro ao processar linha: %v", err)
				continue
			}

			// Converter timestamp para hora local
			formattedTime := time.Unix(timestamp-3*3600, 0).Format("15:04")
			sensorData.Timestamps = append(sensorData.Timestamps, formattedTime)

			// Popular dados com validação de NULL
			addNullable := func(src sql.NullFloat64, dest *[]float64) {
				if src.Valid {
					*dest = append(*dest, src.Float64)
				} else {
					*dest = append(*dest, 0.0)
				}
			}

			addNullable(temp, &sensorData.Temperature)
			addNullable(hum, &sensorData.Humidity)
			addNullable(rain, &sensorData.RainLevel)

			// Converter m/s para km/h
			if wind.Valid {
				sensorData.WindSpeed = append(sensorData.WindSpeed, wind.Float64*3.6)
			} else {
				sensorData.WindSpeed = append(sensorData.WindSpeed, 0.0)
			}
		}

		// Serializar para JSON
		sensorDataJSON, err := json.Marshal(sensorData)
		if err != nil {
			log.Printf("Erro ao serializar JSON: %v", err)
			http.Error(w, "Erro interno", http.StatusInternalServerError)
			return
		}

		// Passar dados para o template
		templates.ExecuteTemplate(w, "dashboard.html", map[string]interface{}{
			"SensorData": template.JS(sensorDataJSON), // Dados completos para gráficos
		})
	}
}

func ApiDashboardHandler(w http.ResponseWriter, r *http.Request) {
	db, err := sql.Open("mysql", DatabaseConfig())
	if err != nil {
		respondWithError(w, "Erro ao conectar ao banco", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// Cálculo de timestamps (mesmo código da Dashboard)
	now := time.Now().UTC()
	localNow := now.Add(-3 * time.Hour)
	startDay := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, time.UTC)
	startTimestamp := startDay.Unix() + 3*3600
	endTimestamp := startTimestamp + 24*3600

	rows, err := db.Query(`
        SELECT timestamp, temperature, humidity, rain_level, average_wind_speed 
        FROM sensor_data 
        WHERE timestamp BETWEEN ? AND ? 
        ORDER BY timestamp
    `, startTimestamp, endTimestamp)

	if err != nil {
		respondWithError(w, "Erro ao buscar dados", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Processamento igual à Dashboard
	var sensorData struct {
		Timestamps  []string  `json:"timestamps"`
		Temperature []float64 `json:"temperature"`
		Humidity    []float64 `json:"humidity"`
		RainLevel   []float64 `json:"rain_level"`
		WindSpeed   []float64 `json:"wind_speed"`
	}

	for rows.Next() {
		var timestamp int64
		var temperature, humidity, rainLevel, windSpeed sql.NullFloat64

		if err := rows.Scan(&timestamp, &temperature, &humidity, &rainLevel, &windSpeed); err != nil {
			log.Printf("Erro ao processar linha: %v", err)
			continue
		}

		formattedTime := time.Unix(timestamp-3*3600, 0).Format("15:04") // UTC-3
		sensorData.Timestamps = append(sensorData.Timestamps, formattedTime)
		if temperature.Valid {
			sensorData.Temperature = append(sensorData.Temperature, temperature.Float64)
		}
		if humidity.Valid {
			sensorData.Humidity = append(sensorData.Humidity, humidity.Float64)
		}
		if rainLevel.Valid {
			sensorData.RainLevel = append(sensorData.RainLevel, rainLevel.Float64)
		}
		if windSpeed.Valid {
			sensorData.WindSpeed = append(sensorData.WindSpeed, windSpeed.Float64*3.6) // m/s para km/h
		}
	}
	// ... (código de processamento igual à Dashboard)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sensorData)
}

func PlotData(templates *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		db, err := sql.Open("mysql", DatabaseConfig())
		if err != nil {
			log.Printf("Erro ao conectar ao banco: %v", err)
			http.Error(w, "erro interno", http.StatusInternalServerError)
			return
		}
		defer db.Close()

		// calculo do inicio e fim do dia (UTC-3)
		now := time.Now().UTC()
		localNow := now.Add(-3 * time.Hour) //UTC-3
		startDay := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, time.UTC)
		startTimeStamp := startDay.Unix() + 3*3600
		endTimeStamp := startTimeStamp + 24*3600

		// Buscar dados no banco
		query := `
			SELECT timestamp, temperature
			FROM sensor_data
			WHERE timestamp BETWEEN ? AND ?
			ORDER BY timestamp
		`
		rows, err := db.Query(query, startTimeStamp, endTimeStamp)
		if err != nil {
			log.Printf("Erro ao conectar ao banco: %v", err)
			http.Error(w, "erro interno", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var timestamps []string
		var temperatures []float64

		for rows.Next() {
			var timestamp int64
			var temperature sql.NullFloat64

			if err := rows.Scan(&timestamp, &temperature); err != nil {
				log.Printf("Erro ao processar linha: %v", err)
				continue
			}

			formattedTime := time.Unix(timestamp-3*3600, 0).Format("15:04") // UTC-3
			timestamps = append(timestamps, formattedTime)
			if temperature.Valid {
				temperatures = append(temperatures, temperature.Float64)
			}
		}

		if len(temperatures) == 0 {
			http.Error(w, "Nenhum dado disponível", http.StatusNotFound)
			return
		}

		sensorData := map[string]interface{}{
			"timestamps":  timestamps,
			"Temperature": temperatures,
		}

		sensorDataJSON, _ := json.Marshal(sensorData)

		// Calcular métricas
		lastTemperature := temperatures[len(temperatures)-1]
		averageTemperature := utils.CalculateAverage(temperatures)
		maxTemperature := utils.CalculateMax(temperatures)
		minTemperature := utils.CalculateMin(temperatures)

		templates.ExecuteTemplate(w, "temp.html", map[string]interface{}{
			"Timestamps":         timestamps,
			"Temperatures":       temperatures,
			"LastTemperature":    lastTemperature,
			"AverageTemperature": averageTemperature,
			"MaxTemperature":     maxTemperature,
			"MinTemperature":     minTemperature,
			"SensorData":         template.JS(sensorDataJSON), // Usar template.JS
		})
	}
}

func ApiTemperatureHandler(w http.ResponseWriter, r *http.Request) {
	db, err := sql.Open("mysql", DatabaseConfig())
	if err != nil {
		log.Printf("Erro ao conectar ao banco: %v", err)
		respondWithError(w, "Erro interno do servidor", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// Cálculo do período (UTC-3)
	now := time.Now().UTC()
	localNow := now.Add(-3 * time.Hour)
	startDay := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, time.UTC)
	startTimestamp := startDay.Unix() + 3*3600
	endTimestamp := startTimestamp + 24*3600

	// Buscar dados
	rows, err := db.Query(`
        SELECT timestamp, temperature 
        FROM sensor_data 
        WHERE timestamp BETWEEN ? AND ? 
        ORDER BY timestamp
    `, startTimestamp, endTimestamp)

	if err != nil {
		log.Printf("Erro na consulta: %v", err)
		respondWithError(w, "Erro ao buscar dados", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Processar resultados
	var response struct {
		Timestamps   []string  `json:"timestamps"`
		Temperatures []float64 `json:"temperatures"`
		Last         float64   `json:"last_temperature"`
		Average      float64   `json:"average_temperature"`
		Max          float64   `json:"max_temperature"`
		Min          float64   `json:"min_temperature"`
		Error        string    `json:"error,omitempty"`
	}

	var temps []float64

	for rows.Next() {
		var timestamp int64
		var temp sql.NullFloat64

		if err := rows.Scan(&timestamp, &temp); err != nil {
			log.Printf("Erro ao ler linha: %v", err)
			continue
		}

		// Converter timestamp para hora local (UTC-3)
		formattedTime := time.Unix(timestamp-3*3600, 0).Format("15:04")
		response.Timestamps = append(response.Timestamps, formattedTime)

		if temp.Valid {
			response.Temperatures = append(response.Temperatures, temp.Float64)
			temps = append(temps, temp.Float64)
		}
	}

	// Verificar dados
	if len(temps) == 0 {
		respondWithError(w, "Nenhum dado de temperatura disponível", http.StatusNotFound)
		return
	}

	// Calcular métricas
	response.Last = temps[len(temps)-1]
	response.Average = utils.CalculateAverage(temps)
	response.Max = utils.CalculateMax(temps)
	response.Min = utils.CalculateMin(temps)

	// Enviar resposta
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Erro ao serializar resposta: %v", err)
		respondWithError(w, "Erro interno", http.StatusInternalServerError)
	}
}
func respondWithError(w http.ResponseWriter, message string, code int) {
	log.Println(message)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
