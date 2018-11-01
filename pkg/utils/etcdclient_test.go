package utils

import "testing"

func TestEtcdSetDefaultHost(t *testing.T) {

	//err := EtcdSetDefaultHost("172.30.60.8:8009||0||{")
	err := EtcdSetDefaultHost("127.0.0.1:50051||0||{")
	if err != nil{
		t.Fatal(err)
	}

}

/*func TestDelGroup(t *testing.T) {
	EtcdDelGroup("ulu_temp_test_group_201711011440")
}*/