package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
)

type APIServer struct {
	ListenAddrServer string
	Store            Storage
}

func NewAPIServer(listenAddrerss string, store Storage) *APIServer {
	return &APIServer{
		ListenAddrServer: listenAddrerss,
		Store:            store,
	}
}

func (a *APIServer) Run() {
	router := mux.NewRouter().StrictSlash(true)

	// Account
	router.HandleFunc("/account", a.CreateHandleAccount).Methods(http.MethodPost)
	router.HandleFunc("/accounts", withJWTAuth(a.GetHandleAccounts, a.Store)).Methods(http.MethodGet)
	router.HandleFunc("/account/{id}", a.DeleteHandleAccount).Methods(http.MethodDelete)
	router.HandleFunc("/account/{id}", a.GetHandleAccountById).Methods(http.MethodGet)

	// Transfer
	router.HandleFunc("/transfer", a.GetTransferAmount).Methods(http.MethodGet)

	// No Routes
	router.NotFoundHandler = http.HandlerFunc(notFoundHandler)

	log.Println("Starting server at: " + a.ListenAddrServer)
	log.Fatal(http.ListenAndServe(a.ListenAddrServer, router))
}

func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(w, "404 Not Found")
}

func WriteJson(w http.ResponseWriter, statusCode int, value any) error {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	return json.NewEncoder(w).Encode(value)
}

func createToken(accountNumaber int32) (string, error) {
	claims := jwt.MapClaims{}
	claims["accountNumaber"] = accountNumaber
	claims["exp"] = time.Now().Add(time.Hour * 24).Unix()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte("secret"))
}

func withJWTAuth(handler http.HandlerFunc, store Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("Calling JWT middleware..")

		vars := mux.Vars(r)
		idStr, found := vars["id"]
		if !found {
			log.Println("withJWTAuth() id is missing")
			WriteJson(w, http.StatusBadRequest, "id is missing")
			return
		}
		id, err := strconv.Atoi(idStr)
		if err != nil {
			log.Println("withJWTAuth() id is not a valid number")
			WriteJson(w, http.StatusBadRequest, "id is not a valid number")
			return
		}

		account, err := store.GetAccountById(id)
		if err != nil {
			log.Println("withJWTAuth() id is not a valid number")
			WriteJson(w, http.StatusBadRequest, "id is not a valid number")
			return
		}

		// JWT
		tokenString := r.Header.Get("x-jwt-token")
		token, err := validateToken(tokenString)
		if err != nil {
			log.Println("withJWTAuth() failed: ", err)
			WriteJson(w, http.StatusUnauthorized, "Failed to authenticate")
			return
		}
		if !token.Valid {
			WriteJson(w, http.StatusUnauthorized, "Failed to authenticate")
			return
		}

		claim := token.Claims.(jwt.MapClaims)
		if account.Number != int32(claim["accountNumber"].(float32)) { // create your own claims structures
			WriteJson(w, http.StatusUnauthorized, "Failed to authenticate")
			return
		}

		handler(w, r)
	}
}

func validateToken(tokenString string) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte("secret"), nil // secret is not secure here..
	})

	return token, err
}

/*
================================
Account Handlers starts here..
================================
*/

// Endpoint: POST /account
func (a *APIServer) CreateHandleAccount(w http.ResponseWriter, r *http.Request) {
	createAccount := &CreateAccountRequest{}
	if err := json.NewDecoder(r.Body).Decode(createAccount); err != nil {
		log.Println("CreateHandleAccount() failed to parse: ", err)
		WriteJson(w, http.StatusInternalServerError, "Failed to parse the json")
		return
	}

	account := NewAccout(createAccount.FirstName, createAccount.LastName)

	jwtToken, err := createToken(account.Number)
	if err != nil {
		log.Println("CreateHandleAccount() failed create token: ", err)
		WriteJson(w, http.StatusInternalServerError, "failed create token")
		return
	}

	fmt.Println("JWT_TOKEN:", jwtToken)

	if err := a.Store.CreateAccount(account); err != nil {
		log.Println("CreateHandleAccount() failed to store: ", err)
		WriteJson(w, http.StatusInternalServerError, "Failed to create a record")
		return
	}

	WriteJson(w, http.StatusOK, account)
}

// Endpoint: GET /accounts
func (a *APIServer) GetHandleAccounts(w http.ResponseWriter, r *http.Request) {
	accounts, err := a.Store.GetAccounts()
	if err != nil {
		log.Println("GetHandleAccounts() failed to get accounts: ", err)
		WriteJson(w, http.StatusInternalServerError, "Failed to get all acounts")
		return
	}

	WriteJson(w, http.StatusOK, accounts)
}

func (a *APIServer) GetHandleAccountById(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr, found := vars["id"]
	if !found {
		log.Println("GetHandleAccountById() id is missing")
		WriteJson(w, http.StatusBadRequest, "id is missing")
		return
	}
	if idStr == "" {
		log.Println("GetHandleAccountById() id is empty")
		WriteJson(w, http.StatusBadRequest, "id is empty")
		return
	}
	id, err := strconv.Atoi(idStr)
	if err != nil {
		log.Println("GetHandleAccountById() id is not a valid number")
		WriteJson(w, http.StatusBadRequest, "id is not a valid number")
		return
	}

	account, err := a.Store.GetAccountById(id)
	if err == sql.ErrNoRows {
		log.Println("GetHandleAccountById() no rows in result set: ", err)
		WriteJson(w, http.StatusNotFound, "no rows in result set")
		return
	}
	if err != nil {
		log.Println("GetHandleAccountById() failed to get account by id: ", err)
		WriteJson(w, http.StatusInternalServerError, "failed to get account by id")
		return
	}

	WriteJson(w, http.StatusOK, account)
}

func (a *APIServer) DeleteHandleAccount(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr, found := vars["id"]
	if !found {
		log.Println("DeleteHandleAccount() id is missing")
		WriteJson(w, http.StatusBadRequest, "id is missing")
		return
	}
	id, err := strconv.Atoi(idStr)
	if err != nil {
		log.Println("DeleteHandleAccount() id is not a valid number")
		WriteJson(w, http.StatusBadRequest, "id is not a valid number")
		return
	}

	if err = a.Store.DeleteAccount(id); err != nil {
		log.Println("DeleteHandleAccount() failed to delete: ", err)
		WriteJson(w, http.StatusInternalServerError, "failed to delete")
		return
	}

	WriteJson(w, http.StatusOK, "success")
}

func (a *APIServer) GetTransferAmount(w http.ResponseWriter, r *http.Request) {
	transferReq := TransferAmmountRequest{}

	if err := json.NewDecoder(r.Body).Decode(&transferReq); err != nil {
		log.Println("handlerTransferAmount() failed to parse: ", err)
		WriteJson(w, http.StatusBadRequest, "failed to parse")
		return
	}

	WriteJson(w, http.StatusOK, transferReq)
}
