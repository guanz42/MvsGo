package mvs

/*
#cgo CFLAGS: -I/opt/MVS/include
#cgo LDFLAGS: -Wl,-rpath=/opt/MVS/lib/64
#cgo LDFLAGS: -L/opt/MVS/lib/64
#cgo LDFLAGS: -lMvCameraControl
#include <stdio.h>
#include <string.h>
#include <unistd.h>
#include <stdlib.h>
#include "MvCameraControl.h"
*/
import "C"

import (
	"errors"
	"fmt"
	"io/ioutil"
	"unsafe"
)

type Mvs struct {
	handle unsafe.Pointer
}

func (mvs *Mvs) GetSDKVersion() string {
	version := uint32(C.MV_CC_GetSDKVersion())

	return fmt.Sprintf("0x%08x", version)
}

// Init 按ModelName初始化相机，ModelName未找到时使用序列号为0的相机
func (mvs *Mvs) Init(modelName string) error {
	var (
		stDeviceList C.MV_CC_DEVICE_INFO_LIST
		stDevInfo    *C.MV_CC_DEVICE_INFO
		index        int
	)

	// enum device
	nRet := C.MV_CC_EnumDevices(C.MV_GIGE_DEVICE|C.MV_USB_DEVICE,
		(*C.MV_CC_DEVICE_INFO_LIST)(unsafe.Pointer(&stDeviceList)))

	if C.MV_OK != int32(nRet) {
		return errors.New(fmt.Sprintf("enum devices fail, %x", uint32(nRet)))
	}

	if 0 >= int32(stDeviceList.nDeviceNum) {
		return errors.New("no camera found")
	}

	fmt.Printf("MV_CC_EnumDevices  nRet=[%x], Count=[%d]\n", int(nRet), uint32(stDeviceList.nDeviceNum))

	for i := 0; i < int(stDeviceList.nDeviceNum); i++ {
		devInfo := stDeviceList.pDeviceInfo[i]
		if devInfo.nTLayerType == C.MV_GIGE_DEVICE {
			specialInfo := *(*C.MV_GIGE_DEVICE_INFO)(unsafe.Pointer(&devInfo.SpecialInfo))
			chModelName := C.GoString((*C.char)(unsafe.Pointer(&specialInfo.chModelName)))
			if modelName == chModelName {
				fmt.Printf("Device Model Name: %s\n", modelName)
				index = i
				break
			}
		} else if devInfo.nTLayerType == C.MV_USB_DEVICE {
			specialInfo := *(*C.MV_USB3_DEVICE_INFO)(unsafe.Pointer(&devInfo.SpecialInfo))
			chModelName := C.GoString((*C.char)(unsafe.Pointer(&specialInfo.chModelName)))
			if modelName == chModelName {
				fmt.Printf("Device Model Name: %s\n", modelName)
				index = i
				break
			}
		} else {
			continue
		}
	}

	stDevInfo = stDeviceList.pDeviceInfo[index]

	// select device and create handle
	nRet = C.MV_CC_CreateHandle(&mvs.handle, stDevInfo)
	if C.MV_OK != int32(nRet) {
		return errors.New(fmt.Sprintf("create handle fail, %x", uint32(nRet)))
	}

	// open device
	nRet = C.MV_CC_OpenDevice(mvs.handle, C.uint(C.MV_ACCESS_Exclusive), C.ushort(0))
	if C.MV_OK != int32(nRet) {
		return errors.New(fmt.Sprintf("open device fail, %x", uint32(nRet)))
	}

	return nil
}

func (mvs *Mvs) StartGrabbing() error {
	// start grab image
	nRet := C.MV_CC_StartGrabbing(mvs.handle)
	if C.MV_OK != int32(nRet) {
		return errors.New(fmt.Sprintf("start grabbing fail, %x", uint32(nRet)))
	}
	return nil
}

func (mvs *Mvs) StopGrabbing() error {
	// end grab image
	nRet := C.MV_CC_StopGrabbing(mvs.handle)
	if C.MV_OK != int32(nRet) {
		return errors.New(fmt.Sprintf("stop grabbing fail, %x", uint32(nRet)))
	}

	return nil
}

func (mvs *Mvs) Cleanup() error {
	// close device
	nRet := C.MV_CC_CloseDevice(mvs.handle)
	if C.MV_OK != int32(nRet) {
		return errors.New(fmt.Sprintf("close device fail, %x", uint32(nRet)))
	}

	// destroy handle
	nRet = C.MV_CC_DestroyHandle(mvs.handle)
	if C.MV_OK != nRet {
		return errors.New(fmt.Sprintf("destroy handle fail, %x", uint32(nRet)))
	}

	return nil

}

func (mvs *Mvs) Capture(filename string) error {
	var (
		stParam     C.MVCC_INTVALUE
		stImageInfo C.MV_FRAME_OUT_INFO_EX
		stSaveParam C.MV_SAVE_IMAGE_PARAM_EX
	)

	//获取一帧数据的大小
	cPayloadSize := C.CString("PayloadSize")
	defer C.free(unsafe.Pointer(cPayloadSize))

	nRet := C.MV_CC_GetIntValue(mvs.handle, cPayloadSize, &stParam)
	if C.MV_OK != nRet {
		return errors.New(fmt.Sprintf("get PayloadSize fail, %x", uint32(nRet)))
	}

	nBufSize := int(stParam.nCurValue) //一帧数据大小
	pFrameBuf := (*C.uchar)(C.malloc(C.size_t(nBufSize)))
	defer C.free(unsafe.Pointer(pFrameBuf))

	nRet = C.MV_CC_GetOneFrameTimeout(mvs.handle, pFrameBuf, C.uint(nBufSize),
		(*C.MV_FRAME_OUT_INFO_EX)(unsafe.Pointer(&stImageInfo)), C.uint(1000))
	if C.MV_OK == int32(nRet) {
		fmt.Printf("GetOneFrame, Width[%d], Height[%d], nFrameNum[%d]\n",
			int(stImageInfo.nWidth), int(stImageInfo.nHeight), int(stImageInfo.nFrameNum))
	} else {
		return errors.New(fmt.Sprintf("no data, %x", uint32(nRet)))
	}

	//源数据
	stSaveParam.pData = pFrameBuf                     //原始图像数据
	stSaveParam.nDataLen = stImageInfo.nFrameLen      //原始图像数据长度
	stSaveParam.enPixelType = stImageInfo.enPixelType //原始图像数据的像素格式
	stSaveParam.nWidth = stImageInfo.nWidth           //图像宽
	stSaveParam.nHeight = stImageInfo.nHeight         //图像高
	stSaveParam.nJpgQuality = C.uint(80)              //图片编码质量

	//目标数据
	nDstBufSize := int(uint(stImageInfo.nWidth)*uint(stImageInfo.nHeight)*4 + 2048)
	pDataForSaveImage := (*C.uchar)(C.malloc(C.size_t(nDstBufSize)))
	defer C.free(unsafe.Pointer(pDataForSaveImage))

	stSaveParam.enImageType = C.MV_Image_Bmp      //需要保存的图像类型
	stSaveParam.nBufferSize = C.uint(nDstBufSize) //存储节点的大小
	stSaveParam.pImageBuffer = pDataForSaveImage  //输出数据缓冲区，存放转换之后的图片数据

	nRet = C.MV_CC_SaveImageEx2(mvs.handle, &stSaveParam)
	if C.MV_OK != nRet {
		return errors.New(fmt.Sprintf("failed in MV_CC_SaveImage, %x", uint32(nRet)))
	}

	dataSlice := C.GoBytes(unsafe.Pointer(pDataForSaveImage), C.int(stSaveParam.nImageLen))
	err := ioutil.WriteFile(filename, dataSlice, 0666)
	if err != nil {
		return errors.New(fmt.Sprintf("write file error, %s", err.Error()))
	}

	return nil
}

func (mvs *Mvs) FeatureSave(filename string) error {
	fName := C.CString(filename)
	defer C.free(unsafe.Pointer(fName))

	nRet := C.MV_CC_FeatureSave(mvs.handle, fName)
	if C.MV_OK != nRet {
		return errors.New(fmt.Sprintf("MV_CC_FeatureSave fail, filename:%s, %x", filename, uint32(nRet)))
	}

	return nil
}

func (mvs *Mvs) FeatureLoad(filename string) error {
	fName := C.CString(filename)
	defer C.free(unsafe.Pointer(fName))

	nRet := C.MV_CC_FeatureLoad(mvs.handle, fName)
	if C.MV_OK != nRet {
		return errors.New(fmt.Sprintf("MV_CC_FeatureLoad fail, filename:%s, %x", filename, uint32(nRet)))
	}

	return nil
}
