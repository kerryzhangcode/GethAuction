package main

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	// "github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/holiman/uint256"
	// "github.com/ethereum/go-ethereum/triedb"
	"main/contract"
)

var (
	// address can't be set as 0-10
    contractAddress = common.HexToAddress("0x00000000000000000000000000000000000000FF")
	AuctionAddress  = common.HexToAddress("0x0000000000000000000000000000000000000100")
	AuctionNFTAddress  = common.HexToAddress("0x0000000000000000000000000000000000000101")
    EOAAddress      = common.HexToAddress("0x0000000000000000000000000000000000000200")
	EOATokenReceiverAddress = common.HexToAddress("0x0000000000000000000000000000000000000201")
	EOAAuction1Address = common.HexToAddress("0x0000000000000000000000000000000000000202")
	EOAAuction2Address = common.HexToAddress("0x0000000000000000000000000000000000000203")
)

// const eth100 = "100000000000000000000"


// MockChainContext 是 ChainContext 的简单实现
type MockChainContext struct{}

func (m *MockChainContext) GetHeader(hash common.Hash, number uint64) *types.Header {
	return nil
}


// 初始化 EVM 执行环境
var statedb *state.StateDB
var evmContext = vm.BlockContext{
	CanTransfer: func(db vm.StateDB, from common.Address, amount *uint256.Int) bool {
		return db.GetBalance(from).Cmp(amount) >= 0 // 检查余额是否足够
	},
	Transfer: func(db vm.StateDB, from common.Address, to common.Address, amount *uint256.Int) {
		db.SubBalance(from, amount, tracing.BalanceChangeUnspecified)
		db.AddBalance(to, amount, tracing.BalanceChangeUnspecified)
	},
	GetHash:     nil,
	Coinbase:    common.HexToAddress("0x0000000000000000000000000000000000000001"), // 矿工地址
	GasLimit:    uint64(10000000),
	BlockNumber: big.NewInt(1),
	Time:        uint64(1672502400),
	Difficulty:  big.NewInt(0),
	BaseFee:     big.NewInt(0),
}
var config = params.AllEthashProtocolChanges
var vmConfig = vm.Config{
	Tracer: &tracing.Hooks{
		OnLog: func(log *types.Log){
			fmt.Printf("Log: %v\n", log)
		},
		// OnOpcode: func(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, rData []byte, depth int, err error) {
		// 	fmt.Printf("Op: %s, Gas: %d, Cost: %d, Depth: %d\n", op, gas, cost, depth)
		// },
		OnTxStart: func(vm *tracing.VMContext, tx *types.Transaction, from common.Address){
			fmt.Printf("Tx Start: %v\n", tx)
			fmt.Printf("From: %v\n", from)
		},
		// OnGasChange: func(old, new uint64, reason tracing.GasChangeReason){
		// 	fmt.Printf("Gas Change: %d -> %d, Reason: %v\n", old, new, reason)
		// },
		OnFault: func(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, depth int, err error){
			fmt.Printf("Fault: %v\n", err)
		},
		OnExit: func(depth int, output []byte, gasUsed uint64, err error, reverted bool){
			fmt.Printf("Exit: Gas used: %d, Reverted: %v\n", gasUsed, reverted)
			// fmt.Printf("Output: %x\n", output)
			fmt.Printf("Error: %v\n", err)
		},
	},
}

func createEvm (address common.Address) *vm.EVM{
	txContext := vm.TxContext{
		Origin:  address,
		GasPrice: big.NewInt(1000),
	}
	evm := vm.NewEVM(evmContext, txContext, statedb, config, vmConfig)
	return evm
}

func main() {
	// 初始化状态数据库
	var err error
	stateDB := state.NewDatabaseForTesting()
	statedb, err = state.New(common.Hash{}, stateDB)
	if err != nil {
		fmt.Printf("StateDB initialization failed: %v\n", err)
		return
	}

	// 读取合约数据
	AuctionContract := contract.GetContracts("Auction")
	AuctionNFTContract := contract.GetContracts("AuctionNFT")

	// 修改db内容
	// eth100 := uint256.NewInt(0).SetFromBig(new(big.Int).SetString("100000000000000000000", 10))
	bigInt, ok := new(big.Int).SetString("100000000000000000000", 10)
	if !ok {
		panic("Invalid number format")
	}
	eth100, _ := uint256.FromBig(bigInt)
	statedb.SetBalance(EOAAddress, eth100, tracing.BalanceChangeUnspecified)
	statedb.SetBalance(EOAAuction1Address, eth100, tracing.BalanceChangeUnspecified)
	statedb.SetBalance(EOAAuction2Address, eth100, tracing.BalanceChangeUnspecified)
	statedb.SetCode(AuctionAddress, common.Hex2Bytes(AuctionContract.Artifact.Bytecode[2:]))

	// 解析 ABI
	auctionParsedABI, err := abi.JSON(strings.NewReader(AuctionContract.ABIJSON))
	if err != nil {
		fmt.Printf("Failed to parse Auction ABI: %v", err)
	}
	auctionNFTParsedABI, err := abi.JSON(strings.NewReader(AuctionNFTContract.ABIJSON))
	if err != nil {
		fmt.Printf("Failed to parse AuctionNFT ABI: %v", err)
	}
	// 输出所有方法
	// for methodName, method := range auctionNFTParsedABI.Methods {
	// 	fmt.Printf("Method: %q, Details: %+v\n", methodName, method)
	// }	

	
	// 创建 EVM
	evmContract := createEvm(EOAAddress)
	evmToken := createEvm(EOATokenReceiverAddress)
	// evmAuction1 := createEvm(EOAAuction1Address)
	// evmAuction2 := createEvm(EOAAuction2Address)
	gas := uint64(300000000000)
	value := uint256.NewInt(0)


	// 部署合约
	sender := vm.AccountRef(contractAddress)
	_, AuctionNFTAddress, gasUsed, err := evmContract.Create(sender, common.Hex2Bytes(AuctionNFTContract.Artifact.Bytecode[2:]), gas, value)
	if err != nil {
		fmt.Printf("AuctionNFT Deployment failed: %v\n", err)
		return
	}
	auctionConstructorArgs := []interface{}{AuctionNFTAddress}
	constructorInput, err := auctionParsedABI.Pack("", auctionConstructorArgs...)
	if err != nil {
		fmt.Printf("Failed to pack constructor function call: %v", err)
	}
	deployAuctionData := append(common.Hex2Bytes(AuctionContract.Artifact.Bytecode[2:]), constructorInput...)
	_, AuctionAddress, gasUsed, err := evmContract.Create(sender, deployAuctionData, gas, value)
	// fmt.Printf("Contract deployed at: %s\n", AuctionNFTAddress.Hex())


	// fmt.Printf("Balance before: %s\n", statedb.GetBalance(EOAAddress))
	// fmt.Printf("Gas before: %d\n", gas)

	// 执行 EVM 代码
	params := []interface{}{EOATokenReceiverAddress, "https://ipfs.io/ipfs/Qm"}
	input, err := auctionNFTParsedABI.Pack("mint", params...)
	if err != nil {
		fmt.Printf("Failed to pack name function call: %v", err)
	}
	result, gasUsed, err := evmContract.Call(sender, AuctionNFTAddress, input, gas, value)
	if err != nil {
		fmt.Printf("Execution failed: %v\n", err)
		return
	}
	// 使用 unpack 解码返回值
	res, err := auctionNFTParsedABI.Unpack("mint", result)
	if err != nil {
		fmt.Printf("Decoding failed\n")
	} else {
		fmt.Printf("Decoded Data: %+v\n", res)
	}
	tokenID := res[0].(*big.Int)

	// 转移到拍卖合约
	params = []interface{}{EOATokenReceiverAddress, AuctionAddress, tokenID}
	input, err = auctionNFTParsedABI.Pack("safeTransferFrom", params...)
	if err != nil {
		fmt.Printf("Failed to pack safeTransferFrom function call: %v", err)
	}
	result, _, err = evmToken.Call(vm.AccountRef(EOATokenReceiverAddress), AuctionNFTAddress, input, gas, value)
	fmt.Printf("result: %x\n", result)
	if err != nil {
		fmt.Printf("Execution failed: %v\n", err)
		return
	}

	// 开始拍卖
	params = []interface{}{tokenID,  big.NewInt(1000), big.NewInt(100), big.NewInt(1672602400)}
	input, _ = auctionParsedABI.Pack("startAuction", params...)
	result, _, _ = evmContract.Call(sender, AuctionAddress, input, gas, value)
	// 使用 unpack 解码返回值
	res, err = auctionParsedABI.Unpack("startAuction", result)
	if err != nil {
		fmt.Printf("Revert data: %x\n", result)
		fmt.Printf("Decoding failed\n")
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Decoded Data: %+v\n", res)
	}
	// auctionRecordId := res[0].(*big.Int)

	// 查看AuctionRecord
	params = []interface{}{tokenID}
	input, _ = auctionParsedABI.Pack("getAuctionRecord", params...)
	result, _, _ = evmContract.Call(sender, AuctionAddress, input, gas, value)
	fmt.Printf("Raw result: %x\n", result)
	fmt.Printf("Outputs: %+v\n", auctionParsedABI.Methods["getAuctionRecord"].Outputs)
	// 使用 unpack 解码返回值
	res, err = auctionParsedABI.Unpack("getAuctionRecord", result)
	if err != nil {
		fmt.Printf("Revert data: %x\n", result)
		fmt.Printf("Decoding failed\n")
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Decoded Data: %+v\n", res)
	}
	

	
	fmt.Printf("Gas used: %d\n", gasUsed)
}
