package main

import (
	"log"
	"os"
	"testing"

	unitTest "github.com/Valiben/gin_unit_test"
	utils "github.com/Valiben/gin_unit_test/utils"
)

func init() {
	router := RouterSetup()
	unitTest.SetRouter(router)
	newLog := log.New(os.Stdout, "\n", log.Llongfile|log.Ldate|log.Ltime)
	unitTest.SetLog(newLog)
}

func TestRootRoute(t *testing.T) {
	resp, err := unitTest.TestOrdinaryHandler(utils.GET, "/", utils.JSON, nil)
	if err != nil {
		t.Errorf("Root Test Error: %v\n", err)
		return
	}
	if string(resp) != "Bitswap Exchange Manager" {
		t.Errorf("Unexpected Response: %v\n", string(resp))
		return
	}
}