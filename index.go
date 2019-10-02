package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"

	mgo "github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/gorilla/mux"
)

type db struct {
	collection *mgo.Collection
}

type payment struct {
	From     int    `bson:"from"`
	To       int    `bson:"to"`
	Amount   int    `bson:"amount"`
	Currency string `bson:"currency"`
}

type paymentsWithUserInfo struct {
	Version  string
	Payments []payment
	User     user
}

type user struct {
	UserID int    `bson:"userid"`
	Name   string `bson:"name"`
}

func (db *db) createPayments(payments ...*payment) {
	for _, payment := range payments {
		err := db.collection.Insert(payment)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func (db *db) getPaymentsByUserID(userID int) *paymentsWithUserInfo {
	var payments []payment
	var payment payment
	items := db.collection.Find(bson.M{"from": userID}).Iter()
	for items.Next(&payment) {
		payments = append(payments, payment)
	}
	url := fmt.Sprintf("http://%s/user/%d", os.Getenv("USERS_SERVICE"), userID)
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	var user user
	content, _ := ioutil.ReadAll(resp.Body)
	err = json.Unmarshal(content, &user)
	return &paymentsWithUserInfo{
		Version:  "v2",
		Payments: payments,
		User:     user,
	}
}

// Ping echoes a Pong message
func Ping(w http.ResponseWriter, r *http.Request) {
	response, _ := json.Marshal("Pong")
	w.Header().Set("Content-Type", "application/json")
	w.Write(response)
}

// GetPaymentsByUserID responds with a JSON of the user with the given id
func (db *db) GetPaymentsByUserID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userIDInt, _ := strconv.Atoi(vars["userid"])
	users := db.getPaymentsByUserID(userIDInt)
	response, _ := json.Marshal(users)
	w.Header().Set("Content-Type", "application/json")
	w.Write(response)
}

func main() {
	session, err := mgo.Dial(os.Getenv("DB_HOST"))
	if err != nil {
		panic(err)
	}
	defer session.Close()

	c := session.DB("paymentsdb").C("payment")
	db := &db{collection: c}

	// If there are no users, create some
	docCount, err := db.collection.Count()
	if docCount == 0 {
		db.createPayments(&payment{
			From:     1,
			To:       2,
			Amount:   100,
			Currency: "$",
		}, &payment{
			From:     1,
			To:       2,
			Amount:   200,
			Currency: "$",
		}, &payment{
			From:     2,
			To:       1,
			Amount:   150,
			Currency: "$",
		})
	}

	r := mux.NewRouter()

	r.HandleFunc(
		"/ping",
		Ping).Methods("GET")

	r.HandleFunc(
		"/payments_from/{userid:[0-9]+}",
		db.GetPaymentsByUserID).Methods("GET")

	srv := &http.Server{
		Handler: r,
		Addr:    "0.0.0.0:8000",
	}
	log.Fatal(srv.ListenAndServe())

}
