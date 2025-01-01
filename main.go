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
)

// MockChainContext 是 ChainContext 的简单实现
type MockChainContext struct{}

func (m *MockChainContext) GetHeader(hash common.Hash, number uint64) *types.Header {
	return nil
}

func main() {
	// 初始化区块链和状态数据库
	config := params.AllEthashProtocolChanges
	stateDB := state.NewDatabaseForTesting()
	statedb, err := state.New(common.Hash{}, stateDB)
	if err != nil {
		fmt.Printf("StateDB initialization failed: %v\n", err)
		return
	}
	// 读取合约数据
	AuctionContract := contract.GetContracts("Auction")
	// 输出 common.Hex2Bytes(AuctionContract.Artifact.Bytecode)?
	// fmt.Printf("Auction ABI: %s\n", AuctionContract.Artifact.Bytecode)
	// fmt.Println(common.Hex2Bytes(AuctionContract.Artifact.Bytecode[2:]))
	AuctionNFTContract := contract.GetContracts("AuctionNFT")

	// 修改db内容
	// code := []byte{0x60, 0x03, 0x60, 0x05, 0x01, 0x60, 0x00, 0x52, 0x60, 0x20, 0x60, 0x00, 0xf3}
	

	statedb.AddBalance(EOAAddress, uint256.NewInt(1000000000000000000), tracing.BalanceChangeUnspecified)
	statedb.SetCode(AuctionAddress, common.Hex2Bytes(AuctionContract.Artifact.Bytecode[2:]))
	// statedb.SetCode(AuctionNFTAddress, common.Hex2Bytes(AuctionNFTContract.Artifact.Bytecode[2:]))
	// fmt.Printf("Stored code: %x\n", statedb.GetCode(AuctionAddress))
	// 解析 ABI
	// auctionParsedABI, err := abi.JSON(strings.NewReader(AuctionContract.ABIJSON))
	// if err != nil {
	// 	fmt.Printf("Failed to parse Auction ABI: %v", err)
	// }
	auctionNFTParsedABI, err := abi.JSON(strings.NewReader(AuctionNFTContract.ABIJSON))
	if err != nil {
		fmt.Printf("Failed to parse AuctionNFT ABI: %v", err)
	}
	// 输出所有方法
	for methodName, method := range auctionNFTParsedABI.Methods {
		fmt.Printf("Method: %q, Details: %+v\n", methodName, method)
	}	
	
	// 检查方法是否存在
	// method := "getBalance" // 替换为实际方法
	// if _, ok := auctionNFTParsedABI.Methods[method]; !ok {
	// 	fmt.Printf("Method '%s' not found in ABI\n", method)
	// 	return
	// }
	// 打包函数调用
	// constructorParams := []interface{}{EOAAddress}
	// constructorData, err := auctionNFTParsedABI.Pack("")
	// if err != nil {
	// 	fmt.Printf("Failed to pack constructor function call: %v", err)
	// }
	// // 合并构造函数参数和字节码
	// AuctionNFTContractCode := append(common.Hex2Bytes(AuctionNFTContract.Artifact.Bytecode[2:]), constructorData...)
	// statedb.SetCode(AuctionNFTAddress, AuctionNFTContractCode)

	evmContext := vm.BlockContext{
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
	txContext := vm.TxContext{
		Origin:  EOAAddress,
		GasPrice: big.NewInt(1000),
	}

	// 创建 EVM
	evm := vm.NewEVM(evmContext, txContext, statedb, config, vm.Config{
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
			// OnExit: func(depth int, output []byte, gasUsed uint64, err error, reverted bool){
			// 	fmt.Printf("Exit: Gas used: %d, Reverted: %v\n", gasUsed, reverted)
			// 	fmt.Printf("Output: %x\n", output)
			// 	fmt.Printf("Error: %v\n", err)
			// },
		},
		
	})
	gas := uint64(300000000000)
	value := uint256.NewInt(0)
	sender := vm.AccountRef(EOAAddress)


	// 部署合约
	_, AuctionNFTAddress, gasUsed, err := evm.Create(sender, common.Hex2Bytes(AuctionNFTContract.Artifact.Bytecode[2:]), gas, value)
	if err != nil {
		fmt.Printf("Deployment failed: %v\n", err)
		return
	}
	fmt.Printf("Contract deployed at: %s\n", AuctionNFTAddress.Hex())


	fmt.Printf("Balance before: %s\n", statedb.GetBalance(EOAAddress))
	fmt.Printf("Gas before: %d\n", gas)
	// code := statedb.GetCode(AuctionNFTAddress)
	// fmt.Printf("Code: %x\n", code)

	// 执行 EVM 代码
	params := []interface{}{EOATokenReceiverAddress, "https://ipfs.io/ipfs/Qm"}
	// params := []interface{}{}
	input, err := auctionNFTParsedABI.Pack("mint", params...)
	if err != nil {
		fmt.Printf("Failed to pack name function call: %v", err)
	}
	// sender := vm.AccountRef(EOAAddress)
	result, gasUsed, err := evm.Call(sender, AuctionNFTAddress, input, gas, value)
	if err != nil {
		fmt.Printf("Execution failed: %v\n", err)
		return
	}
	

	fmt.Printf("Balance after: %s\n", statedb.GetBalance(EOAAddress))
	// fmt.Printf("Execution result: %x\n", result)
	// result 是字节数组，需要手动解码字符串
	// decodedString := string(result)
	// fmt.Printf("Decoded String: %s\n", decodedString)
	// 使用 unpack 解码返回值
	res, err := auctionNFTParsedABI.Unpack("mint", result)
	if err != nil {
		fmt.Printf("Decoding failed\n")
	} else {
		fmt.Printf("Decoded Data: %+v\n", res)
	}
	fmt.Printf("Gas used: %d\n", gasUsed)
}
