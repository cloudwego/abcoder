package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

// prettyPrintJSON 读取 JSON 文件，格式化后写回原文件
func prettyPrintJSON(filePath string) error {
	// 读取文件内容
	fileContent, err := ioutil.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("读取文件失败: %w", err)
	}

	var data interface{}
	// 解析 JSON 数据
	err = json.Unmarshal(fileContent, &data)
	if err != nil {
		return fmt.Errorf("解析 JSON 数据失败: %w", err)
	}

	// 格式化 JSON 数据
	prettyJSON, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return fmt.Errorf("格式化 JSON 数据失败: %w", err)
	}

	// 将格式化后的 JSON 数据写回原文件
	err = ioutil.WriteFile(filePath, prettyJSON, 0644)
	if err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}

	return nil
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("用法: go run pretty_json.go <json_file_path>")
		os.Exit(1)
	}

	filePath := os.Args[1]
	err := prettyPrintJSON(filePath)
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("JSON 文件 %s 已成功格式化。\n", filePath)
}
