package utils

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/go-playground/validator"
)

const (
	B  = 1
	KB = 1 << (iota * 10)
	MB
	GB
	TB
	PB
	EB
	ZB
	YB
	BB
)

func Validate(objects ...interface{}) error {

	validate := validator.New()

	for _, obj := range objects {
		err := validate.Struct(obj)
		if err != nil {
			return err
		}
	}

	return nil
}

func ParseDataSize(dataSize string) (int, error) {

	symbol := strings.ToUpper(dataSize[(len(dataSize) - 2):])
	number := dataSize[:(len(dataSize) - 2)]

	value, err := strconv.Atoi(number)
	if err != nil {
		return 0, err
	}

	switch symbol {
	case "MB":
		return value * MB, nil
	case "GB":
		return value * GB, nil
	case "TB":
		return value * TB, nil
	case "PB":
		//TODO: find a solution for bigger numbers here
		if value > 9000 {
			return 0, fmt.Errorf("number too big %dPB to bytes it might cause overflow", value)
		}
		return value * PB, nil
		// case "EB":
		// 	return value * EB, nil
		// TODO overflow
		// case "ZB":
		// 	return value * ZB, nil
		// case "YB":
		// 	return value * YB, nil
		// case "BB":
		// 	return value * BB, nil
	}

	return 0, nil
}
