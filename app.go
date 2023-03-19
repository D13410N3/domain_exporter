package main

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "log"
    "net/http"
    "os"
    "os/exec"
    "strings"
    "time"
    "unicode"

    "github.com/gorilla/mux"
    "github.com/joho/godotenv"
)

type Zone struct {
    ID     string `json:"id"`
    Name   string `json:"name"`
    Status string `json:"status"`
}

type ApiResponse struct {
    Result []Zone `json:"result"`
}

func main() {
    godotenv.Load()

    listenAddr := os.Getenv("LISTEN_ADDR")
    if listenAddr == "" {
        listenAddr = "127.0.0.1"
    }

    listenPort := os.Getenv("LISTEN_PORT")
    if listenPort == "" {
        listenPort = "9988"
    }

    cfToken := os.Getenv("CF_TOKEN")
    if cfToken == "" {
        log.Fatal("CF_TOKEN environment variable is not set")
    }

    router := mux.NewRouter()
    router.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "text/plain")
        w.WriteHeader(http.StatusOK)

        zones, err := fetchZones(cfToken)
        if err != nil {
            log.Printf("Failed to fetch zones: %v\n", err)
            return
        }

        for _, zone := range zones {
            if !isLatinDomain(zone.Name) {
                continue
            }
            expiryTime, err := getExpiryTime(zone.Name)
            if err != nil {
                log.Printf("Failed to get expiry time for domain %s: %v\n", zone.Name, err)
                continue
            }
            fmt.Fprintf(w, "domain_expiry_time{domain=\"%s\"} %d\n", zone.Name, expiryTime)
        }
    })

    log.Printf("Starting web server on %s:%s\n", listenAddr, listenPort)
    log.Fatal(http.ListenAndServe(listenAddr+":"+listenPort, router))
}


func fetchZones(cfToken string) ([]Zone, error) {
    client := &http.Client{}
    req, err := http.NewRequest("GET", "https://api.cloudflare.com/client/v4/zones?per_page=50", nil)
    if err != nil {
        return nil, err
    }

    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+cfToken)

    resp, err := client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }

    var apiResponse ApiResponse
    err = json.Unmarshal(body, &apiResponse)
    if err != nil {
        return nil, err
    }

    return apiResponse.Result, nil
}

func isLatinDomain(domain string) bool {
    for _, r := range domain {
        if r > unicode.MaxLatin1 {
            return false
        }
    }
    return true
}

func getExpiryTime(domain string) (int64, error) {
    tld := strings.Split(domain, ".")[1]

    var whoisCmd string
    switch tld {
    case "ru":
        whoisCmd = "whois " + domain + " | grep 'paid-till' | awk '{print $2}'"
    default:
        whoisCmd = "whois " + domain + " | grep 'Registry Expiry Date' | awk '{print $NF}'"
    }

    out, err := exec.Command("bash", "-c", whoisCmd).Output()
    if err != nil {
        return 0, err
    }

    expiryDate, err := time.Parse(time.RFC3339, strings.TrimSpace(string(out)))
    if err != nil {
        return 0, err
    }

    time.Sleep(1 * time.Second)

    return expiryDate.Unix(), nil
}

