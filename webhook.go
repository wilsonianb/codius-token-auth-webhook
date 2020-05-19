package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	authv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func authenticate(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.NotFound(rw, req)
		return
	}
	body := json.NewDecoder(req.Body)
	tr := &authv1.TokenReview{}
	err := body.Decode(tr)
	if err != nil {
		handleErr(rw, err)
		return
	}

	req = &http.Request{
		Header: map[string][]string{},
	}
	url := fmt.Sprintf("%s/balances/%s:spend", os.Getenv("RECEIPT_VERIFIER_URL"), tr.Spec.Token)
	log.Println(url)
	resp, err := http.Post(url, "text/plain", bytes.NewBuffer([]byte(os.Getenv("AUTH_PRICE"))))
	if err != nil {
		fmt.Println("Balance spend error:", err)
		handleErr(rw, err)
		return
	}
	b, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		fmt.Println("Balance spend error:", string(b))
		handleErr(rw, errors.New(string(b)))
		return
	}
	fmt.Println("Balance:", string(b))

	user := os.Getenv("RBAC_USER")
	// groups := []string{
	// 	"testgroup",
	// }
	trResp := &authv1.TokenReview{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "authentication.k8s.io/v1",
			Kind:       "TokenReview",
		},
		Status: authv1.TokenReviewStatus{
			Authenticated: true,
			User: authv1.UserInfo{
				UID:      user,
				Username: user,
				// Groups:   groups,
			},
		},
	}

	writeResp(rw, trResp)
}

func writeResp(rw http.ResponseWriter, tr *authv1.TokenReview) {
	rw.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(rw)
	err := enc.Encode(tr)
	if err != nil {
		log.Println("Failed to encode token review response")
	}
}

func handleErr(rw http.ResponseWriter, err error) {
	writeResp(rw, &authv1.TokenReview{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "authentication.k8s.io/v1",
			Kind:       "TokenReview",
		},
		Status: authv1.TokenReviewStatus{
			Error: err.Error(),
		},
	})
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := fmt.Sprintf(":%s", port) 
	http.HandleFunc("/authenticate", authenticate)
	fmt.Println("Starting server on", addr)
	http.ListenAndServe(addr, nil)
}
