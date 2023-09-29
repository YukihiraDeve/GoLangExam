package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const ip = "10.49.122.144"
const timeout = 3 * time.Second

// findOpenPort scanne les ports et retourne le premier port ouvert qui répond avec 200 OK à /ping
func findOpenPort() (int, string, error) {
	var wg sync.WaitGroup
	var foundPort int
	var pingResponse string
	client := &http.Client{Timeout: timeout}

	for port := 5174; port <= 5174; port++ {
		wg.Add(1)
		go func(p int) {
			defer wg.Done()
			portStr := strconv.Itoa(p)
			conn, err := net.DialTimeout("tcp", net.JoinHostPort(ip, portStr), timeout)
			if err != nil {
				return
			}
			conn.Close()
			url := fmt.Sprintf("http://%s:%d/ping", ip, p)
			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
			if err != nil {
				return
			}
			resp, err := client.Do(req)
			if err != nil {
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				foundPort = p
				body, err := ioutil.ReadAll(resp.Body)
				if err == nil {
					pingResponse = string(body)
				}
			}
		}(port)
	}
	wg.Wait()

	if foundPort == 0 {
		return 0, "", fmt.Errorf("aucun port ouvert trouvé répondant avec 200 OK à /ping")
	}
	return foundPort, pingResponse, nil
}

// postRequest effectue une requête POST avec les données fournies et retourne la réponse
func postRequest(client *http.Client, url string, data map[string]string) (*http.Response, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la création du JSON : %w", err)
	}
	resp, err := client.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la requête POST sur %s: %w", url, err)
	}
	return resp, nil
}

func main() {
	port, pingResponse, err := findOpenPort()
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("Port trouvé: %d\nRéponse à /ping: %s\n", port, pingResponse)
	client := &http.Client{Timeout: timeout}

	userData := map[string]string{"User": "Valentin"}

	signupURL := fmt.Sprintf("http://%s:%d/signup", ip, port)
	checkURL := fmt.Sprintf("http://%s:%d/check", ip, port)

	_, err = postRequest(client, signupURL, userData)
	if err != nil {
		fmt.Printf("Erreur lors de la requête POST sur %s: %s\n", signupURL, err)
		return
	}

	_, err = postRequest(client, checkURL, userData)
	if err != nil {
		fmt.Printf("Erreur lors de la requête POST sur %s: %s\n", checkURL, err)
		return
	}

	secretURL := fmt.Sprintf("http://%s:%d/getUserSecret", ip, port)
	for i := 0; i < 100; i++ {
		resp, err := postRequest(client, secretURL, userData)
		if err != nil {
			fmt.Printf("Erreur lors de la requête POST sur %s: %s\n", secretURL, err)
			continue
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("Erreur lors de la lecture du corps de la réponse sur %s: %s\n", secretURL, err)
			continue
		}

		secret := string(body)
		fmt.Printf("Tentative %d, Réponse: %s\n", i+1, secret)
		if strings.HasPrefix(secret, "User secret:") {
			fmt.Println("Clé secrète trouvée:", secret)
			trimmedSecretKey := strings.TrimSpace(strings.TrimPrefix(secret, "User secret:"))
			userData["secret"] = trimmedSecretKey // Mettre à jour le secret dans userData
			break
		}
	}

	// Essayer différents chemins avec le secret trouvé
	paths := []string{"/getUserLevel", "/getUserPoints", "/iNeedAHint", "/enterChallenge", "/submitSolution"}
	for _, path := range paths {

		url := fmt.Sprintf("http://%s:%d%s", ip, port, path)
		resp, err := postRequest(client, url, userData)
		if err != nil {
			fmt.Printf("Erreur lors de la requête POST sur %s: %s\n", url, err)
			continue
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("Erreur lors de la lecture du corps de la réponse sur %s: %s\n", url, err)
			continue
		}
		if path == "/getUserLevel" {
			levelStr := strings.TrimSpace(string(body))
			userData["Level"] = strings.TrimPrefix(levelStr, "Level: ")
		}
		if path == "/getUserPoints" {
			pointsStr := strings.TrimSpace(string(body))
			parts := strings.Split(pointsStr, " ")
			if len(parts) > 0 {
				userData["Points"] = parts[len(parts)-1]
			}
		}

		if path == "/submitSolution" {
			userData["Protocol"] = "MD5"

			submitSolutionData := map[string]interface{}{
				"User":   userData["User"],
				"Secret": userData["secret"],
				"Content": map[string]interface{}{
					"Level": userData["Level"],
					"Challenge": map[string]string{
						"Username": userData["User"],
						"Secret":   "393f0e0270b94d13b1610db79f878423",
						"Points":   userData["Points"],
					},
					"Protocol":  userData["Protocol"],
					"SecretKey": "4A1EAF7A5881",
				},
			}

			fmt.Println(userData["User"])
			fmt.Println(userData["secret"])
			fmt.Println(userData["Level"])
			fmt.Println(userData["Points"])
			fmt.Println(userData["Protocol"])

			jsonData, err := json.Marshal(submitSolutionData)
			if err != nil {
				fmt.Printf("Erreur lors de la création du JSON : %s\n", err)
				return
			}

			resp, err := client.Post(url, "application/json", bytes.NewBuffer(jsonData))
			if err != nil {
				fmt.Printf("Erreur lors de la requête POST sur %s: %s\n", url, err)
				return
			}
			defer resp.Body.Close()

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				fmt.Printf("Erreur lors de la lecture du corps de la réponse sur %s: %s\n", url, err)
				return
			}

			// Afficher la réponse pour /submitSolution
			fmt.Printf("Réponse POST de %s: %s\nCorps: %s\n", url, resp.Status, string(body))
			continue
		}

		// Afficher les en-têtes et le corps de la réponse pour chaque chemin
		fmt.Printf("Corps: %s\n", string(body))
	}
}
