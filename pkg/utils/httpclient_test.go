package utils

import (
	"testing"
	"encoding/base64"
	"io/ioutil"
	"fmt"
	"os"
	"encoding/json"
	"errors"
	"github.com/widaT/facegetway/utils"
)

func TestDistance(t *testing.T)   {



	fmt.Println(similarity(1.2))
	fmt.Println(similarity2(1.2))
}



func TestGetFeature(t *testing.T) {
	ff, err := ioutil.ReadFile("/home/wida/face/0a9b4f8f7be69ed891a95e54d4cb2024/0a9b4f8f7be69ed891a95e54d4cb2024_0_2918809.jpg")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	str := base64.StdEncoding.EncodeToString(ff)
	feature,err := GetFeature(Conf.GetString("http","feature"),str)
	if err != nil {
		t.Fatal(err)
	}
	length := len(feature)
	if length != 512 {
		t.Fatalf(" 512  %d",length )
	}
	fmt.Println(feature)
}

func TestDetect(t *testing.T){
	ff, err := ioutil.ReadFile("/home/wida/test/2460920_0_1931_1246311019_318_face_alarm.jpg")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	str := base64.StdEncoding.EncodeToString(ff)

	ret,err := Detect(Conf.GetString("http","detect"),str)
	if err != nil {
		t.Fatal(err)
	}


	js,_ := json.Marshal(ret)



	fmt.Println(string(js))
}

func TestSimilarity(t *testing.T)  {
	ff, err := ioutil.ReadFile("/home/wida/test/2460920_0_1930_1246311015_943_face_alarm.jpg")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	str := base64.StdEncoding.EncodeToString(ff)
	feature1,err := GetFeature(Conf.GetString("http","feature"),str)


	fmt.Println(feature1)

	ff, err = ioutil.ReadFile("/home/wida/test")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	str = base64.StdEncoding.EncodeToString(ff)
	feature2,err := GetFeature(Conf.GetString("http","feature"),str)


	a ,_:= euclidean(feature1,feature2)


	fmt.Println(a)
	fmt.Println(similarity(a))
	fmt.Println(similarity2(a))
}


func euclidean(infoA, infoB []FeatureFloat) (FeatureFloat, error) {
	if len(infoA) != len(infoB) {
		return 0, errors.New("params err")
	}
	var distance FeatureFloat
	for i, number := range infoA {
		a := number - infoB[i]
		distance += a * a
	}
	return distance, nil
}


func similarity(diff FeatureFloat) FeatureFloat {
	threshold :=FeatureFloat(utils.Conf.GetFloat32("base","threshold"))
	maxDiff := FeatureFloat(0.6*threshold + threshold)
	similarity := FeatureFloat(0.0)
	if diff > maxDiff {
		similarity = 0.0
	} else {
		similarity = (maxDiff - diff) / threshold
		if similarity > 1.0 {
			similarity = 1.0
		}
	}
	return similarity * 100
}


func similarity2(distance FeatureFloat) FeatureFloat {
	similarity := FeatureFloat(0.0)
	if distance > 2.65 {
		similarity = 0.0
	} else {
		similarity = (2.65 - distance) / 2.65 * 100
	}
	return similarity
}