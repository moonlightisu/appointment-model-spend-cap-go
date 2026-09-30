package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

type appointment struct {
	Reference string `json:"reference"`
	StartsAt  string `json:"starts_at"`
	State     string `json:"state"`
}

func notificationInput(a appointment) (string, error) {
	if a.Reference == "" || len(a.Reference) > 64 {
		return "", fmt.Errorf("reference must contain 1-64 characters")
	}
	start, err := time.Parse(time.RFC3339, a.StartsAt)
	if err != nil {
		return "", fmt.Errorf("starts_at must be RFC3339")
	}
	verb := map[string]string{"confirmed": "confirmed", "rescheduled": "rescheduled", "cancelled": "cancelled"}[a.State]
	if verb == "" {
		return "", fmt.Errorf("state must be confirmed, rescheduled or cancelled")
	}
	return fmt.Sprintf("Draft a short operational appointment notice. Reference %q; appointment %s; state %s. Mention no diagnosis, treatment, test result, or patient name. Do not invent clinical details.", a.Reference, start.Format(time.RFC3339), verb), nil
}

func main() {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		log.Fatal("set INFRAI_API_KEY")
	}
	capUSD, err := strconv.ParseFloat(os.Getenv("MONTHLY_CAP_USD"), 64)
	if err != nil || capUSD <= 0 {
		log.Fatal("set positive MONTHLY_CAP_USD")
	}
	account := accountClient{key: key, http: &http.Client{Timeout: 15 * time.Second}}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := account.setBudget(ctx, capUSD); err != nil {
		log.Fatal(err)
	}
	ai := openai.NewClient(option.WithAPIKey(key), option.WithBaseURL(baseURL))
	mux := http.NewServeMux()
	mux.HandleFunc("POST /appointment-notices", func(w http.ResponseWriter, r *http.Request) {
		var a appointment
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&a); err != nil {
			http.Error(w, "invalid appointment JSON", http.StatusBadRequest)
			return
		}
		prompt, err := notificationInput(a)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		// One account credential applies the cap and pays for this inference call.
		completion, err := ai.Chat.Completions.New(r.Context(), openai.ChatCompletionNewParams{
			Model:    "auto",
			Messages: []openai.ChatCompletionMessageParamUnion{openai.UserMessage(prompt)},
		})
		if err != nil {
			http.Error(w, "notification generation unavailable", http.StatusBadGateway)
			return
		}
		if len(completion.Choices) == 0 {
			http.Error(w, "empty notification", http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"reference": a.Reference, "notice": completion.Choices[0].Message.Content})
	})
	mux.HandleFunc("GET /usage-timeseries", func(w http.ResponseWriter, r *http.Request) {
		data, err := account.usageTimeseries(r.Context())
		if err != nil {
			http.Error(w, "usage query unavailable", http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	})
	log.Fatal(http.ListenAndServe("127.0.0.1:8080", mux))
}
