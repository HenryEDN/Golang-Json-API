package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	// "reflect"
	"strconv"

	jwt "github.com/golang-jwt/jwt/v4"
	"github.com/gorilla/mux"
)

type APIServer struct{
	listenAddr string
	store Storage
}

func NewAPIServer(listenAddr string, store Storage) *APIServer{
	return &APIServer{
		listenAddr: listenAddr,
		store: store,
	}
}

func (s *APIServer) Run(){
	router := mux.NewRouter()

	router.HandleFunc("/account", makeHTTPHandleFunc(s.handleAccount))
	router.HandleFunc("/account/{id}", withJWTAuth(makeHTTPHandleFunc(s.handleGetAccountByID), s.store))
	router.HandleFunc("/transfer", makeHTTPHandleFunc(s.handleTransfer))

	log.Println("JSON API server running on port: ", s.listenAddr)

	http.ListenAndServe(s.listenAddr, router)
}

func (s *APIServer) handleGetAccount(w http.ResponseWriter, r *http.Request) error{
	accounts, err := s.store.GetAccounts()
	if err != nil{
		return err
	}

	return WriteJSON(w, http.StatusOK, accounts)
}

func (s *APIServer) handleAccount(w http.ResponseWriter, r *http.Request) error{

	switch r.Method{
		case "GET":
			return s.handleGetAccount(w, r)
		case "POST":
			return s.handleCreateAccount(w, r)
		case "DELETE":
			return s.handleDeleteAccount(w, r)
		}
	return fmt.Errorf("method not allowed %s", r.Method)
}

func (s *APIServer) handleGetAccountByID(w http.ResponseWriter, r *http.Request) error{
	if r.Method == "GET"{
		id, err := getID(r)
		if err != nil{
			return err
		}

		account, err := s.store.GetAccountById(id)
		if err != nil{
			return err
		}

		return WriteJSON(w, http.StatusOK, account)
	}

	if r.Method == "DELETE"{
		return s.handleDeleteAccount(w, r)
	}

	return fmt.Errorf("method not allowed %s", r.Method)
}

func (s *APIServer) handleCreateAccount(w http.ResponseWriter, r *http.Request) error{
	createAccountReq := new(CreateAccountRequest)
	if err := json.NewDecoder(r.Body).Decode(&createAccountReq); err != nil{
		return err
	}
	
	account := NewAccount(createAccountReq.FirstName, createAccountReq.LastName)

	if err := s.store.CreateAccount(account); err != nil{
		return err
	}

	tokenString, err := createJWT(account)
	if err != nil{
		return err
	}

	fmt.Println("JWT token: ", tokenString)

	return WriteJSON(w, http.StatusOK, account)
}

func (s *APIServer) handleDeleteAccount(w http.ResponseWriter, r *http.Request) error{
	id, err := getID(r)
	if err != nil{
		return err
	}

	if err := s.store.DeleteAccount(id); err != nil{
		return err
	}
	return WriteJSON(w, http.StatusOK, map[string]int{"deleted": id})
}

func (s *APIServer) handleTransfer(w http.ResponseWriter, r *http.Request) error{
	transferRequest := new(TransferRequest)
	if err := json.NewDecoder(r.Body).Decode(transferRequest); err != nil{
		return err
	}
	defer r.Body.Close()

	return WriteJSON(w, http.StatusOK, transferRequest)
}

func WriteJSON(w http.ResponseWriter, status int, v any) error{
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}

func permissionDenied(w http.ResponseWriter){
	WriteJSON(w, http.StatusForbidden, ApiError{Error: "permission denied"})
}

// eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhY2NvdW50TnVtYmVyIjo2ODYwNSwiZXhwaXJlc0F0IjoxNTAwMH0.KYjIKzws7z02jP4hbeMqGj3TpHSY9m4Hs-lJTuTKrT0

func withJWTAuth(handlerFunc http.HandlerFunc, s Storage) http.HandlerFunc{
	return func(w http.ResponseWriter, r *http.Request){
		fmt.Println("calling JWT auth middleware")

		tokenString := r.Header.Get("x-jwt-token")
		token, err := validateJWT(tokenString)

		if err != nil{
			permissionDenied(w)
			return
		}

		if !token.Valid{
			permissionDenied(w)
		}

		userID, err := getID(r)
		if err != nil{
			permissionDenied(w)
			return
		}
		
		account, err := s.GetAccountById(userID)
		if err != nil{
			WriteJSON(w, http.StatusForbidden, ApiError{Error: "invalid token"})
			return
		}

		claims := token.Claims.(jwt.MapClaims)
		// fmt.Println(reflect.TypeOf(claims["accountNumber"]))
		if account.Number != claims["accountNumber"]{
			if err != nil{
				permissionDenied(w)
				return
			}
		}

		handlerFunc(w,r)
	}
}

func validateJWT(tokenString string) (*jwt.Token, error){
	secret := os.Getenv("JWT_SECRET")
	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok{
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(secret), nil
	})
}

func createJWT(account *Account) (string, error){
	// Create the Claims
	claims := &jwt.MapClaims{
		"expiresAt": 15000,
		"accountNumber": account.Number,
	}

	secret := os.Getenv("JWT_SECRET")
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(secret))
}

type apiFunc func(http.ResponseWriter, *http.Request) error

type ApiError struct{
	Error string `json:"error"`
}

func makeHTTPHandleFunc(f apiFunc) http.HandlerFunc{
	return func(w http.ResponseWriter, r *http.Request){
		if err := f(w,r); err != nil{
			WriteJSON(w, http.StatusBadRequest, ApiError{Error: err.Error()})
		}
	}
}

func getID(r *http.Request) (int, error){
	idStr := mux.Vars(r)["id"]
	id, err := strconv.Atoi(idStr)

	if err != nil{
		return id, fmt.Errorf("invalid id given %s", idStr)
	}

	return id, nil
}