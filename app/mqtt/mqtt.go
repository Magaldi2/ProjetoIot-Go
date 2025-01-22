package mqtt

import (
	"database/sql"
	"encoding/json"
	"log"
	"math"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	_ "github.com/go-sql-driver/mysql"
)

// SensorData representa os dados do sensor
type SensorData struct {
	RainLevel        float64 `json:"rain_level"`
	AverageWindSpeed float64 `json:"average_wind_speed"`
	WindDirection    float64 `json:"wind_direction"`
	Humidity         float64 `json:"humidity"`
	UVIndex          float64 `json:"uv_index"`
	SolarRadiation   float64 `json:"solar_radiation"`
	Temperature      float64 `json:"temperature"`
	Timestamp        int64   `json:"timestamp"`
}

// saveToMySQL salva os dados no banco de dados MySQL
func saveToMySQL(data SensorData) {
	db, err := sql.Open("mysql", "root:example@tcp(mysql:3306)/weather_data")
	if err != nil {
		log.Fatalf("Erro ao conectar ao MySQL: %v", err)
	}
	defer db.Close()

	query := `
		INSERT INTO sensor_data (
			rain_level, average_wind_speed, wind_direction, 
			humidity, uv_index, solar_radiation, temperature, timestamp
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE 
			rain_level=VALUES(rain_level),
			average_wind_speed=VALUES(average_wind_speed),
			wind_direction=VALUES(wind_direction),
			humidity=VALUES(humidity),
			uv_index=VALUES(uv_index),
			solar_radiation=VALUES(solar_radiation),
			temperature=VALUES(temperature)
	`
	_, err = db.Exec(query, data.RainLevel, data.AverageWindSpeed, data.WindDirection,
		data.Humidity, data.UVIndex, data.SolarRadiation, data.Temperature, data.Timestamp)
	if err != nil {
		log.Printf("Erro ao salvar dados no MySQL: %v", err)
	} else {
		log.Println("Dados salvos no MySQL:", data)
	}
}

// mqttMessageHandler processa mensagens recebidas
func mqttMessageHandler(client mqtt.Client, msg mqtt.Message) {
	var payload []map[string]interface{}
	err := json.Unmarshal(msg.Payload(), &payload)
	if err != nil {
		log.Printf("Erro ao decodificar JSON: %v", err)
		return
	}

	data := SensorData{Timestamp: time.Now().Unix()}
	for _, item := range payload {
		label := item["n"].(string)
		value := item["v"].(float64)

		switch label {
		case "emw_rain_level":
			data.RainLevel = value
		case "emw_average_wind_speed":
			data.AverageWindSpeed = value
		case "emw_wind_direction":
			data.WindDirection = value
		case "emw_humidity":
			data.Humidity = value
		case "emw_uv":
			data.UVIndex = value
		case "emw_solar_radiation":
			data.SolarRadiation = value
		case "emw_temperature":
			data.Temperature = value
		}
	}

	saveToMySQL(data)
}

func SetupMQTT() {
	opts := mqtt.NewClientOptions()
	opts.AddBroker("tcp://98.84.130.156:1883")
	opts.SetClientID("GoMQTTClient")

	// 1️⃣ Remove log.Fatalf para evitar encerrar o processo
	opts.OnConnect = func(c mqtt.Client) {
		log.Println("Conectado ao broker MQTT!")
		if token := c.Subscribe("konda", 0, mqttMessageHandler); token.Wait() && token.Error() != nil {
			log.Printf("Erro na inscrição: %v", token.Error()) // Só loga, não encerra
			return
		}
	}

	// 2️⃣ Adiciona tentativa de reconexão automática
	opts.OnConnectionLost = func(c mqtt.Client, err error) {
		log.Printf("Conexão perdida: %v", err)
		go reconnectMQTT(c) // Inicia reconexão em background
	}

	client := mqtt.NewClient(opts)

	// 3️⃣ Conexão inicial sem fatal error
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Printf("Falha inicial: %v", token.Error())
		go reconnectMQTT(client) // Começa tentativas de reconexão
	}

	// 4️⃣ Mantém a função viva sem bloquear execução
	for {
		time.Sleep(1 * time.Hour) // Reduz consumo de CPU
	}
}

// 5️⃣ Lógica inteligente de reconexão com backoff
func reconnectMQTT(c mqtt.Client) {
	retryInterval := 5 * time.Second
	for {
		time.Sleep(retryInterval)
		if token := c.Connect(); token.Wait() && token.Error() == nil {
			return // Reconectou com sucesso
		}
		retryInterval = time.Duration(math.Min(float64(retryInterval*2), 300)) // Limita a 5 minutos
	}
}
