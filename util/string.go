package util

import (
	"fmt"
	"github.com/eiannone/keyboard"
	"math/rand"
	"reflect"
	"strconv"
	"time"
)

const (
	// RED 红色
	RED = "\033[31m"
	// GREEN 绿色
	GREEN = "\033[32m"
	// YELLOW 黄色
	YELLOW = "\033[33m"
	// BLUE 蓝色
	BLUE = "\033[34m"
	// FUCHSIA 紫红色
	FUCHSIA = "\033[35m"
	// CYAN 青色
	CYAN = "\033[36m"
	// WHITE 白色
	WHITE = "\033[37m"
	// RESET 重置颜色
	RESET = "\033[0m"
)

// IsInteger 判断字符串是否为整数
func IsInteger(input string) bool {
	_, err := strconv.Atoi(input)
	return err == nil
}

// RandString 随机字符串
func RandString(length int) string {
	var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, length)
	rand.Seed(time.Now().UnixNano())
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func getChar(str string) string {
	err := keyboard.Open()
	if err != nil {
		panic(err)
	}
	defer keyboard.Close()
	fmt.Print(str)
	char, _, _ := keyboard.GetKey()
	fmt.Printf("%c\n", char)
	if char == 0 {
		return ""
	}
	return string(char)
}

// LoopInput 循环输入选择, 或者直接回车退出
func LoopInput(tip string, choices interface{}, print bool) int {
	reflectValue := reflect.ValueOf(choices)
	if reflectValue.Kind() != reflect.Slice && reflectValue.Kind() != reflect.Array {
		fmt.Println("only support slice or array type!")
		return -1
	}
	length := reflectValue.Len()
	if print && reflectValue.Type().String() == "[]string" {
		for i := 0; i < length; i++ {
			fmt.Printf("%d.%s\n\n", i+1, reflectValue.Index(i).Interface())
		}
	}
	for {
		inputString := ""
		if length < 10 {
			inputString = getChar(tip)
		} else {
			fmt.Print(tip)
			_, _ = fmt.Scanln(&inputString)
		}
		if inputString == "" {
			return -1
		} else if !IsInteger(inputString) {
			fmt.Println("输入有误,请重新输入")
			continue
		}
		number, _ := strconv.Atoi(inputString)
		if number <= length && number > 0 {
			return number
		}
		fmt.Println("输入数字越界,请重新输入")
	}
}

// Input 读取终端用户输入
func Input(tip string, defaultValue string) string {
	input := ""
	fmt.Print(tip)
	_, _ = fmt.Scanln(&input)
	if input == "" && defaultValue != "" {
		input = defaultValue
	}
	return input
}

// Red 红色
func Red(str string) string {
	return RED + str + RESET
}

// Green 绿色
func Green(str string) string {
	return GREEN + str + RESET
}

// Yellow 黄色
func Yellow(str string) string {
	return YELLOW + str + RESET
}

// Blue 蓝色
func Blue(str string) string {
	return BLUE + str + RESET
}

// Fuchsia 紫红色
func Fuchsia(str string) string {
	return FUCHSIA + str + RESET
}

// Cyan 青色
func Cyan(str string) string {
	return CYAN + str + RESET
}

// White 白色
func White(str string) string {
	return WHITE + str + RESET
}
