package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
)

func do_login(url string, wg *sync.WaitGroup, out chan<- string, errChan chan<- string) {
	defer wg.Done()
	var jsonStr = []byte(`{"user": "command","password": "command","manId": 1001,"userGroupName": "Administrator|G"}`)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	if err != nil {
		errChan <- fmt.Sprintf("could not build request: %s", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		errChan <- fmt.Sprintf("could not do request: %s", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errChan <- fmt.Sprintf("login response is not status ok: %d\n", resp.StatusCode)
		return
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		errChan <- fmt.Sprintf("could not read all bytes from body: %s", err)
		return
	}
	respmap := make(map[string]interface{})
	err = json.Unmarshal(bodyBytes, &respmap)
	if err != nil {
		errChan <- fmt.Sprintf("could not Unmarshal: %s", string(bodyBytes[:]))
		return
	}
	r, ok := respmap["sessionId"].(string)
	if !ok {
		errChan <- fmt.Sprintf("could not convert sessionId to string")
		return
	}
	out<- r
}

func do_logout(url string, sessionId string, wg *sync.WaitGroup) {
	defer wg.Done()
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		fmt.Printf("could not build request: %s", err) 
		return
	}

	params := req.URL.Query()
	params.Add("sessionId", sessionId)
	req.URL.RawQuery = params.Encode()

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("could not do request: %s", err) 
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("logout response is not status ok: %d\n", resp.StatusCode)
		return
	}
}

func main () {
	sessions := make(chan string)
	errors := make(chan string)
	var loginWg sync.WaitGroup
	for i := 0; i < 200; i++ {
		loginWg.Add(1)
		go do_login("http://localhost:8080/app/command/axis/api/rest/businessGateway/login", &loginWg, sessions, errors)
	}
	go func() {
		loginWg.Wait()
		close(sessions)
	}()
	var logoutWg sync.WaitGroup
	loop: for {
		select {
		case sessionId, ok := <-sessions:
			if !ok {
				break loop
			}
			logoutWg.Add(1)
			go do_logout("http://localhost:8081/app/command/axis/api/rest/businessGateway/logout", sessionId, &logoutWg)
			break
		case errMsg := <-errors:
			fmt.Println(errMsg)
		}
	}
}
