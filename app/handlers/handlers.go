package handlers

import (
	"database/sql"
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
		db, err := sql.Open("mysql", DatabaseConfig())
		if err != nil {
			log.Printf("Erro ao conectar ao banco: %v", err)
			http.Error(w, "Erro interno", http.StatusInternalServerError)
			return
		}
		defer db.Close()

		// Buscar dados
		currentData, previousData := utils.GetMySQLData(db)

		// Preparar dados para o template
		if currentData == nil {
			templates.ExecuteTemplate(w, "index.html", map[string]interface{}{
				"Message": "Nenhum dado disponível no momento.",
			})
			return
		}

		context := utils.PrepareTemplateData(currentData, previousData)
		templates.ExecuteTemplate(w, "index.html", context)
	}
}

func Dashboard(templates *template.Template) http.HandlerFunc{
	return func(w http.ResponseWriter, r *http.Request){
		db, err := sql.Open("mysql", DatabaseConfig())
		if err != nil {
			log.Printf("Erro ao conectar ao banco: %v", err)
			http.Error(w,"erro interno", http.StatusInternalServerError)
			return
		}
		defer db.Close()

		// calculo do inicio e fim do dia (UTC-3)
		now := time.Now().UTC()
		localNow := now.Add(-3 *time.Hour) //UTC-3
		startDay := time.Date(localNow.Year(),localNow.Month(), localNow.Day(), 0,0,0,0,time.UTC)
		startTimestamp := startDay.Unix() + 3*3600
		endTimestamp := startTimestamp + 24*3600

		// Buscar dados do dia atual
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

		sensorData := struct {
			Timestamps   []string
			Temperature  []float64
			Humidity     []float64
			RainLevel    []float64
			WindSpeed    []float64
		}{}

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

		if len(sensorData.Timestamps) == 0 {
			templates.ExecuteTemplate(w, "dashboard.html", map[string]string{
				"Message": "Nenhum dado disponível no momento.",
			})
			return
		}

		templates.ExecuteTemplate(w, "dashboard.html", sensorData)
	}
}

func PlotData(templates *template.Template) http.HandlerFunc{
	return func(w http.ResponseWriter, r *http.Request){
		db, err := sql.Open("mysql", DatabaseConfig())
		if err != nil {
			log.Printf("Erro ao conectar ao banco: %v", err)
			http.Error(w,"erro interno", http.StatusInternalServerError)
			return
		}
		defer db.Close()

		// calculo do inicio e fim do dia (UTC-3)
		now := time.Now().UTC()
		localNow := now.Add(-3 *time.Hour) //UTC-3
		startDay := time.Date(localNow.Year(),localNow.Month(), localNow.Day(), 0,0,0,0,time.UTC)
		startTimeStamp := startDay.Unix() + 3*3600
		endTimeStamp := startTimeStamp + 24*3600

		// Buscar dados no banco
		query := `
			SELECT timestamp, temperature
			FROM sensor_data
			WHERE timestamp BETWEEN ? AND ?
			ORDER BY timestamp
		`
		rows, err := db.Query(query, startTimeStamp , endTimeStamp)
		if err != nil {
			log.Printf("Erro ao conectar ao banco: %v", err)
			http.Error(w,"erro interno", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var timestamps []string
		var temperatures []float64

		for rows.Next(){
			var timestamp int64
			var temperature sql.NullFloat64

			if err:= rows.Scan(&timestamp, &temperature); err != nil {
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
	
		// Calcular métricas
		lastTemperature := temperatures[len(temperatures)-1]
		averageTemperature := utils.CalculateAverage(temperatures)
		maxTemperature := utils.CalculateMax(temperatures)
		minTemperature := utils.CalculateMin(temperatures)

		templates.ExecuteTemplate(w, "temp.html", map[string]interface{}{
			"Timestamps":        timestamps,
			"Temperatures":      temperatures,
			"LastTemperature":   lastTemperature,
			"AverageTemperature": averageTemperature,
			"MaxTemperature":    maxTemperature,
			"MinTemperature":    minTemperature,
		})
	}
}