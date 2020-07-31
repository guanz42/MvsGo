package main

import (
	"fmt"
	"os"
	"time"

	"MvsGo/mvs"
)

func main() {
	cam := new(mvs.Mvs)

	ver := cam.GetSDKVersion()
	fmt.Printf("SDK Version: %s\n", ver)

	err := cam.Init("MV-CE013-50G")
	if err != nil {
		fmt.Println("error: " + err.Error())
		return
	}

	if err := cam.FeatureSave("feature.ini"); err != nil {
		fmt.Println("error: " + err.Error())
		return
	}

	if err := cam.FeatureLoad("feature.ini"); err != nil {
		fmt.Println("error: " + err.Error())
		return
	}

	if err := cam.StartGrabbing(); err != nil {
		fmt.Println("error: " + err.Error())
		return
	}

	_ = os.MkdirAll("images", os.ModePerm)
	captureID := 0
	ticker := time.NewTicker(time.Second * 3)
	timer := time.NewTimer(time.Minute * 1)
	for {
		select {
		case <-ticker.C:
			filename := fmt.Sprintf("images/%d.bmp", captureID)
			if err := cam.Capture(filename); err != nil {
				fmt.Println("error: " + err.Error())
			}
			captureID += 1
		case <-timer.C:
			fmt.Println("time up")
			ticker.Stop()
			if err := cam.StopGrabbing(); err != nil {
				fmt.Println("error: " + err.Error())
			}
			if err := cam.Cleanup(); err != nil {
				fmt.Println("error: " + err.Error())
			}
			return
		}
	}
}
