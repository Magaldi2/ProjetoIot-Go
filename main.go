package main

import (
	"html/template"
	"log"
	"net/http"
	"projeto/app/handlers"
	"projeto/app/mqtt"
)

var templates = template.Must(template.ParseGlob("templates/*.html"))

func main() {
	go mqtt.SetupMQTT()

	// Rotas de templates
	http.HandleFunc("/", handlers.Index(templates))
	http.HandleFunc("/dados", handlers.Dashboard(templates))
	http.HandleFunc("/temperatura", handlers.PlotData(templates))

	// Novas rotas da API
	http.HandleFunc("/api", handlers.ApiIndexHandler)
	http.HandleFunc("/api/dados", handlers.ApiDashboardHandler)
	http.HandleFunc("/api/temperatura", handlers.ApiTemperatureHandler)

	log.Println("Servidor rodando na porta 8080")
	http.ListenAndServe(":8080", nil)
}
