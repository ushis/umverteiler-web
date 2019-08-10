package main

import (
	"crypto/subtle"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/ammario/paypal-ipn"
	"github.com/gobuffalo/packr"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Config struct {
	DB             string
	Listen         string
	AuthToken      string
	PaypalEmail    string
	PaypalCurrency string
}

func main() {
	logger := log.New(os.Stdout, "", log.Lshortfile)

	config, err := loadConfig()

	if err != nil {
		logger.Fatal(err)
	}
	db, err := NewDB(config.DB)

	if err != nil {
		logger.Fatal(err)
	}

	apiHandler := ApiHandler{
		db:          db,
		logger:      logger,
		authToken:   config.AuthToken,
		ipnBusiness: config.PaypalEmail,
		ipnCurrency: config.PaypalCurrency,
	}

	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(packr.NewBox("./static")))
	mux.Handle("/api/", http.StripPrefix("/api", apiHandler.Handler()))

	logger.Printf("listening on http://%s/\n", config.Listen)

	if err := http.ListenAndServe(config.Listen, mux); err != nil {
		logger.Fatal(err)
	}
}

func loadConfig() (Config, error) {
	config := Config{}
	flag.StringVar(&config.DB, "db", "./transactions.csv", "transactions database")
	flag.StringVar(&config.Listen, "listen", "127.0.0.1:8080", "address to listen to")
	flag.StringVar(&config.AuthToken, "auth-token", "", "api authentication token")
	flag.StringVar(&config.PaypalEmail, "paypal-email", "", "paypal email")
	flag.StringVar(&config.PaypalCurrency, "paypal-currency", "", "paypal currency")
	flag.Parse()

	if val := os.Getenv("UMVERTEILER_WEB_DB"); val != "" {
		config.DB = val
	}

	if val := os.Getenv("UMVERTEILER_WEB_LISTEN"); val != "" {
		config.Listen = val
	}

	if val := os.Getenv("UMVERTEILER_WEB_AUTH_TOKEN"); val != "" {
		config.AuthToken = val
	}

	if val := os.Getenv("UMVERTEILER_WEB_PAYPAL_EMAIL"); val != "" {
		config.PaypalEmail = val
	}

	if val := os.Getenv("UMVERTEILER_WEB_PAYPAL_CURRENCY"); val != "" {
		config.PaypalCurrency = val
	}

	if config.AuthToken == "" {
		return config, fmt.Errorf("no -auth-token specified")
	}

	if config.PaypalEmail == "" {
		return config, fmt.Errorf("no -paypal-email specified")
	}

	if config.PaypalCurrency == "" {
		return config, fmt.Errorf("no -paypal-currency specified")
	}
	return config, nil
}

type ApiHandler struct {
	db          *DB
	logger      *log.Logger
	authToken   string
	ipnBusiness string
	ipnCurrency string
}

func NewApiHandler(db *DB, logger *log.Logger, authToken string) ApiHandler {
	return ApiHandler{db: db, logger: logger, authToken: authToken}
}

func (h ApiHandler) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/balance", h.getBalance)
	mux.HandleFunc("/transactions", h.authorize(h.postTransaction))
	mux.HandleFunc("/ipn", ipn.Listener(h.handleIpn))
	return mux
}

func (h ApiHandler) authorize(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		auth := strings.Fields(req.Header.Get("Authorization"))

		if len(auth) != 2 || !h.correctAuthToken(auth[1]) {
			w.Header().Set("WWW-Authenticate", "Bearer")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		next(w, req)
	}
}

func (h ApiHandler) correctAuthToken(token string) bool {
	return subtle.ConstantTimeCompare([]byte(h.authToken), []byte(token)) == 1
}

func (h ApiHandler) getBalance(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(h.db.Balance())
}

func (h ApiHandler) postTransaction(w http.ResponseWriter, req *http.Request) {
	tx := Transaction{}

	if json.NewDecoder(req.Body).Decode(&tx) != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	tx.Comment = strings.TrimSpace(tx.Comment)

	if tx.Amount == 0.0 || tx.Comment == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := h.db.Append(tx); err != nil {
		h.logger.Println("could not save transaction:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	h.logger.Println("append transaction:", tx)
	w.WriteHeader(http.StatusCreated)
}

func (h ApiHandler) handleIpn(err error, n *ipn.Notification) {
	if err != nil {
		h.logger.Println("ipn failed:", err)
		return
	}

	if n.Business != h.ipnBusiness {
		h.logger.Println("ipn failed: bad business:", n.Business)
		return
	}

	if n.Currency != h.ipnCurrency {
		h.logger.Println("ipn failed: bad currency:", n.Currency)
		return
	}

	if n.TestIPN {
		h.logger.Println("ipn test: not saving transaction")
		return
	}

	txGross := Transaction{
		Amount:  n.Gross,
		Date:    *n.PaymentDate.Time,
		Comment: "Paypal Spende",
	}

	txFee := Transaction{
		Amount:  -n.Fee,
		Date:    *n.PaymentDate.Time,
		Comment: "Paypal Gebühr",
	}

	if err := h.db.Append(txGross, txFee); err != nil {
		h.logger.Println("could not save transactions:", err)
		return
	}
	h.logger.Println("append transaction:", txGross)
	h.logger.Println("append transaction:", txFee)
}

type DB struct {
	path    string
	mutex   sync.RWMutex
	balance Balance
}

type Transaction struct {
	Amount  float64   `json:"amount"`
	Date    time.Time `json:"date"`
	Comment string    `json:"comment"`
}

func (tx Transaction) String() string {
	return fmt.Sprintf("%f (%s, %s)", tx.Amount, tx.Date, tx.Comment)
}

type Balance struct {
	Balance float64 `json:"balance"`
}

func NewDB(path string) (*DB, error) {
	db := &DB{path: path, balance: Balance{}}
	err := db.load()
	return db, err
}

func (db *DB) Balance() Balance {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	return db.balance
}

func (db *DB) Append(txs ...Transaction) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	f, err := os.OpenFile(db.path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)

	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)

	for _, tx := range txs {
		row := []string{
			strconv.FormatFloat(tx.Amount, 'f', -1, 64),
			tx.Date.Format(time.RFC3339),
			tx.Comment,
		}

		if err := w.Write(row); err != nil {
			return err
		}
		db.balance.Balance += tx.Amount
	}
	w.Flush()

	return w.Error()
}

func (db *DB) load() error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	db.balance.Balance = 0

	f, err := os.OpenFile(db.path, os.O_RDONLY|os.O_CREATE, 0644)

	if err != nil {
		return err
	}
	defer f.Close()

	r := csv.NewReader(f)

	for {
		row, err := r.Read()

		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}

		if len(row) != 3 {
			return fmt.Errorf("invalid format: number of columns: %d", len(row))
		}
		amount, err := strconv.ParseFloat(row[0], 64)

		if err != nil {
			return fmt.Errorf("invalid format: %s", err)
		}
		db.balance.Balance += amount
	}
	return nil
}
