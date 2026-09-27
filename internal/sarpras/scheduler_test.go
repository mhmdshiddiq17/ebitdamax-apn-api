package sarpras

import (
	"log"
	"testing"
)

func TestStartSchedulerRejectsInvalidInterval(t *testing.T) {
	if _, err := StartScheduler(nil, nil, "bukan-jadwal", log.Default()); err == nil {
		t.Fatal("interval tidak valid harus ditolak")
	}
}
