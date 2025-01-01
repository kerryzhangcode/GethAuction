package contract

import (
	"encoding/json"
	// "fmt"
	"io/ioutil"
	"log"
	"os"
)

// 定义结构体与 JSON 数据结构对应
type Input struct {
	InternalType string `json:"internalType"`
	Name         string `json:"name"`
	Type         string `json:"type"`
}

type Output struct {
	InternalType string `json:"internalType"`
	Name         string `json:"name"`
	Type         string `json:"type"`
}

type ABI struct {
	Inputs          []Input `json:"inputs"`
	Outputs         []Output `json:"outputs"`
	Name			string  `json:"name"`
	StateMutability string  `json:"stateMutability"`
	Type            string  `json:"type"`
}

type ContractArtifact struct {
	Format                string            `json:"_format"`
	ContractName          string            `json:"contractName"`
	SourceName            string            `json:"sourceName"`
	ABI                   []ABI             `json:"abi"`
	Bytecode              string            `json:"bytecode"`
	DeployedBytecode      string            `json:"deployedBytecode"`
	LinkReferences        map[string]string `json:"linkReferences"`
	DeployedLinkReferences map[string]string `json:"deployedLinkReferences"`
}

type Contract struct  {
	Artifact ContractArtifact
	ABIJSON string
}

func GetContracts(conrtactName string)  Contract {
	// 打开 JSON 文件
	file, err := os.Open("./contracts/" + conrtactName +".json")
	if err != nil {
		log.Fatalf("Failed to open file: %v", err)
	}
	defer file.Close()

	// 读取文件内容
	data, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatalf("Failed to read file: %v", err)
	}

	// 定义用于解析的结构体实例
	var artifact ContractArtifact
	err = json.Unmarshal(data, &artifact)
	if err != nil {
		log.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// 将 ABI 转换为 JSON 字符串
	abiJSON, err := json.Marshal(artifact.ABI)
	if err != nil {
		log.Fatalf("Failed to marshal ABI to JSON: %v", err)
	}
	

	// 输出解析结果
	// fmt.Printf("Format: %s\n", artifact.Format)
	// fmt.Printf("Contract Name: %s\n", artifact.ContractName)
	// fmt.Printf("Source Name: %s\n", artifact.SourceName)
	// fmt.Printf("ABI: %+v\n", artifact.ABI)
	// fmt.Printf("Bytecode: %s\n", artifact.Bytecode[:10]) // 只显示前 10 个字符
	// fmt.Printf("Deployed Bytecode: %s\n", artifact.DeployedBytecode[:10]) // 只显示前 10 个字符
	// fmt.Printf("Link References: %+v\n", artifact.LinkReferences)
	// fmt.Printf("Deployed Link References: %+v\n", artifact.DeployedLinkReferences)
	return Contract{
		Artifact: artifact,
		ABIJSON: string(abiJSON),
	}
}