package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Pega a versão da imagem de uma variável de ambiente (boa prática).
		// A variável de ambiente será definida no seu Helm Chart.
		appVersion := os.Getenv("APP_VERSION")
		if appVersion == "" {
			appVersion = "desconhecida"
		}

		// Cria a estrutura da resposta.
		response := map[string]string{
			"serviço":  "sample-api",
			"mensagem": "Olá, OpsMaster!",
			"versão":   appVersion,
		}

		// Define o cabeçalho como JSON.
		w.Header().Set("Content-Type", "application/json")
		// Codifica e envia a resposta.
		json.NewEncoder(w).Encode(response)
	})

	log.Println("Servidor iniciado na porta 8080")
	// Inicia o servidor na porta 8080.
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Falha ao iniciar o servidor: %v", err)
	}
}
