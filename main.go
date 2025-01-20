package main

import (
    "html/template"
    "log"
    "net/http"
    "projeto/app/handlers" // Importando o pacote de handlers
    "projeto/app/mqtt"     // Importando o pacote de MQTT
)

var templates = template.Must(template.ParseGlob("templates/*.html"))

func main() {
    // Iniciar MQTT em uma goroutine
    go mqtt.SetupMQTT()

    // Configurar rotas
    http.HandleFunc("/", handlers.Index(templates))
    http.HandleFunc("/dados", handlers.Dashboard(templates))
    http.HandleFunc("/temperatura", handlers.PlotData(templates))
    
    // Iniciar servidor HTTP
    log.Println("Servidor iniciado na porta 8080")
    if err := http.ListenAndServe(":8080", nil); err != nil {
        log.Fatalf("Erro ao iniciar o servidor: %v", err)
    }
}
