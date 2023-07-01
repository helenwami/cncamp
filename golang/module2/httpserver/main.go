package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func healthz(w http.ResponseWriter, r *http.Request) {

	for k, v := range r.Header {
		w.Write([]byte(k + "\n"))
		w.Write([]byte(v[0] + "\n"))
	}

	httpCode := "200"
	w.Write([]byte(httpCode + "\n"))

	os.Setenv("VERSION", "go1.20.5")
	w.Write([]byte(os.Getenv("VERSION") + "\n"))

	fmt.Fprintln(w, "Hello World!")

	ip := r.RemoteAddr
	code := http.StatusOK
	log.Println(ip)
	log.Println(http.StatusOK)
	fmt.Printf("Request ip addr is: %s \n", ip)
	fmt.Printf("HTTP Status Code is: %d \n", code)

}

func main() {
	http.HandleFunc("/", healthz)
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}
}
