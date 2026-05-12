// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package contract

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
	_ = abi.ConvertType
)

// LeanIMTPlusRootStorageMetaData contains all meta data concerning the LeanIMTPlusRootStorage contract.
var LeanIMTPlusRootStorageMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_relayer\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"issuerId\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"root\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"crlNumber\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint8\",\"name\":\"depth\",\"type\":\"uint8\"},{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"leafCount\",\"type\":\"uint64\"}],\"name\":\"RootUpdated\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"issuerId\",\"type\":\"bytes32\"}],\"name\":\"getRoot\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"issuerId\",\"type\":\"bytes32\"}],\"name\":\"getRootInfo\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"root\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"crlNumber\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"updatedAt\",\"type\":\"uint256\"},{\"internalType\":\"uint8\",\"name\":\"depth\",\"type\":\"uint8\"},{\"internalType\":\"uint64\",\"name\":\"leafCount\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"relayer\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"roots\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"root\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"crlNumber\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"updatedAt\",\"type\":\"uint256\"},{\"internalType\":\"uint8\",\"name\":\"depth\",\"type\":\"uint8\"},{\"internalType\":\"uint64\",\"name\":\"leafCount\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"issuerId\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"newRoot\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"crlNumber\",\"type\":\"uint256\"},{\"internalType\":\"uint8\",\"name\":\"depth\",\"type\":\"uint8\"},{\"internalType\":\"uint64\",\"name\":\"leafCount\",\"type\":\"uint64\"}],\"name\":\"setRoot\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x6080604052348015600e575f5ffd5b50604051610452380380610452833981016040819052602b91604e565b5f80546001600160a01b0319166001600160a01b03929092169190911790556079565b5f60208284031215605d575f5ffd5b81516001600160a01b03811681146072575f5ffd5b9392505050565b6103cc806100865f395ff3fe608060405234801561000f575f5ffd5b5060043610610055575f3560e01c806341cc5bcb146100595780638406c079146100de57806384f9422114610108578063ae6dead714610135578063b688289a1461017e575b5f5ffd5b6100a461006736600461031c565b5f90815260016020819052604090912080549181015460028201546003909201549293909260ff81169161010090910467ffffffffffffffff1690565b6040805195865260208601949094529284019190915260ff16606083015267ffffffffffffffff16608082015260a0015b60405180910390f35b5f546100f0906001600160a01b031681565b6040516001600160a01b0390911681526020016100d5565b61012761011636600461031c565b5f9081526001602052604090205490565b6040519081526020016100d5565b6100a461014336600461031c565b600160208190525f9182526040909120805491810154600282015460039092015490919060ff811690610100900467ffffffffffffffff1685565b61019161018c366004610333565b610193565b005b5f546001600160a01b031633146101e05760405162461bcd60e51b815260206004820152600c60248201526b1d5b985d5d1a1bdc9a5e995960a21b60448201526064015b60405180910390fd5b5f8581526001602081905260409091200154831161022c5760405162461bcd60e51b81526020600482015260096024820152681cdd185b194810d49360ba1b60448201526064016101d7565b6040805160a08101825285815260208082018681524283850190815260ff8088166060860190815267ffffffffffffffff808916608088019081525f8e8152600197889052899020975188559451958701959095559151600286015590516003909401805492519093166101000268ffffffffffffffffff199092169316929092179190911790555185907fecccef0c55a00dc23786b6780a5499d379220ff8f6f9a14c408f0b6546f007359061030d908790879087908790938452602084019290925260ff16604083015267ffffffffffffffff16606082015260800190565b60405180910390a25050505050565b5f6020828403121561032c575f5ffd5b5035919050565b5f5f5f5f5f60a08688031215610347575f5ffd5b853594506020860135935060408601359250606086013560ff8116811461036c575f5ffd5b9150608086013567ffffffffffffffff81168114610388575f5ffd5b80915050929550929590935056fea2646970667358221220ba8ad11652df9518ca435937e141d2e48db897882cc7344ed1427140814385a364736f6c634300081c0033",
}

// LeanIMTPlusRootStorageABI is the input ABI used to generate the binding from.
// Deprecated: Use LeanIMTPlusRootStorageMetaData.ABI instead.
var LeanIMTPlusRootStorageABI = LeanIMTPlusRootStorageMetaData.ABI

// LeanIMTPlusRootStorageBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use LeanIMTPlusRootStorageMetaData.Bin instead.
var LeanIMTPlusRootStorageBin = LeanIMTPlusRootStorageMetaData.Bin

// DeployLeanIMTPlusRootStorage deploys a new Ethereum contract, binding an instance of LeanIMTPlusRootStorage to it.
func DeployLeanIMTPlusRootStorage(auth *bind.TransactOpts, backend bind.ContractBackend, _relayer common.Address) (common.Address, *types.Transaction, *LeanIMTPlusRootStorage, error) {
	parsed, err := LeanIMTPlusRootStorageMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(LeanIMTPlusRootStorageBin), backend, _relayer)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &LeanIMTPlusRootStorage{LeanIMTPlusRootStorageCaller: LeanIMTPlusRootStorageCaller{contract: contract}, LeanIMTPlusRootStorageTransactor: LeanIMTPlusRootStorageTransactor{contract: contract}, LeanIMTPlusRootStorageFilterer: LeanIMTPlusRootStorageFilterer{contract: contract}}, nil
}

// LeanIMTPlusRootStorage is an auto generated Go binding around an Ethereum contract.
type LeanIMTPlusRootStorage struct {
	LeanIMTPlusRootStorageCaller     // Read-only binding to the contract
	LeanIMTPlusRootStorageTransactor // Write-only binding to the contract
	LeanIMTPlusRootStorageFilterer   // Log filterer for contract events
}

// LeanIMTPlusRootStorageCaller is an auto generated read-only Go binding around an Ethereum contract.
type LeanIMTPlusRootStorageCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// LeanIMTPlusRootStorageTransactor is an auto generated write-only Go binding around an Ethereum contract.
type LeanIMTPlusRootStorageTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// LeanIMTPlusRootStorageFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type LeanIMTPlusRootStorageFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// LeanIMTPlusRootStorageSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type LeanIMTPlusRootStorageSession struct {
	Contract     *LeanIMTPlusRootStorage // Generic contract binding to set the session for
	CallOpts     bind.CallOpts           // Call options to use throughout this session
	TransactOpts bind.TransactOpts       // Transaction auth options to use throughout this session
}

// LeanIMTPlusRootStorageCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type LeanIMTPlusRootStorageCallerSession struct {
	Contract *LeanIMTPlusRootStorageCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                 // Call options to use throughout this session
}

// LeanIMTPlusRootStorageTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type LeanIMTPlusRootStorageTransactorSession struct {
	Contract     *LeanIMTPlusRootStorageTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                 // Transaction auth options to use throughout this session
}

// LeanIMTPlusRootStorageRaw is an auto generated low-level Go binding around an Ethereum contract.
type LeanIMTPlusRootStorageRaw struct {
	Contract *LeanIMTPlusRootStorage // Generic contract binding to access the raw methods on
}

// LeanIMTPlusRootStorageCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type LeanIMTPlusRootStorageCallerRaw struct {
	Contract *LeanIMTPlusRootStorageCaller // Generic read-only contract binding to access the raw methods on
}

// LeanIMTPlusRootStorageTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type LeanIMTPlusRootStorageTransactorRaw struct {
	Contract *LeanIMTPlusRootStorageTransactor // Generic write-only contract binding to access the raw methods on
}

// NewLeanIMTPlusRootStorage creates a new instance of LeanIMTPlusRootStorage, bound to a specific deployed contract.
func NewLeanIMTPlusRootStorage(address common.Address, backend bind.ContractBackend) (*LeanIMTPlusRootStorage, error) {
	contract, err := bindLeanIMTPlusRootStorage(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &LeanIMTPlusRootStorage{LeanIMTPlusRootStorageCaller: LeanIMTPlusRootStorageCaller{contract: contract}, LeanIMTPlusRootStorageTransactor: LeanIMTPlusRootStorageTransactor{contract: contract}, LeanIMTPlusRootStorageFilterer: LeanIMTPlusRootStorageFilterer{contract: contract}}, nil
}

// NewLeanIMTPlusRootStorageCaller creates a new read-only instance of LeanIMTPlusRootStorage, bound to a specific deployed contract.
func NewLeanIMTPlusRootStorageCaller(address common.Address, caller bind.ContractCaller) (*LeanIMTPlusRootStorageCaller, error) {
	contract, err := bindLeanIMTPlusRootStorage(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &LeanIMTPlusRootStorageCaller{contract: contract}, nil
}

// NewLeanIMTPlusRootStorageTransactor creates a new write-only instance of LeanIMTPlusRootStorage, bound to a specific deployed contract.
func NewLeanIMTPlusRootStorageTransactor(address common.Address, transactor bind.ContractTransactor) (*LeanIMTPlusRootStorageTransactor, error) {
	contract, err := bindLeanIMTPlusRootStorage(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &LeanIMTPlusRootStorageTransactor{contract: contract}, nil
}

// NewLeanIMTPlusRootStorageFilterer creates a new log filterer instance of LeanIMTPlusRootStorage, bound to a specific deployed contract.
func NewLeanIMTPlusRootStorageFilterer(address common.Address, filterer bind.ContractFilterer) (*LeanIMTPlusRootStorageFilterer, error) {
	contract, err := bindLeanIMTPlusRootStorage(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &LeanIMTPlusRootStorageFilterer{contract: contract}, nil
}

// bindLeanIMTPlusRootStorage binds a generic wrapper to an already deployed contract.
func bindLeanIMTPlusRootStorage(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := LeanIMTPlusRootStorageMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_LeanIMTPlusRootStorage *LeanIMTPlusRootStorageRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _LeanIMTPlusRootStorage.Contract.LeanIMTPlusRootStorageCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_LeanIMTPlusRootStorage *LeanIMTPlusRootStorageRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _LeanIMTPlusRootStorage.Contract.LeanIMTPlusRootStorageTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_LeanIMTPlusRootStorage *LeanIMTPlusRootStorageRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _LeanIMTPlusRootStorage.Contract.LeanIMTPlusRootStorageTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_LeanIMTPlusRootStorage *LeanIMTPlusRootStorageCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _LeanIMTPlusRootStorage.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_LeanIMTPlusRootStorage *LeanIMTPlusRootStorageTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _LeanIMTPlusRootStorage.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_LeanIMTPlusRootStorage *LeanIMTPlusRootStorageTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _LeanIMTPlusRootStorage.Contract.contract.Transact(opts, method, params...)
}

// GetRoot is a free data retrieval call binding the contract method 0x84f94221.
//
// Solidity: function getRoot(bytes32 issuerId) view returns(uint256)
func (_LeanIMTPlusRootStorage *LeanIMTPlusRootStorageCaller) GetRoot(opts *bind.CallOpts, issuerId [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _LeanIMTPlusRootStorage.contract.Call(opts, &out, "getRoot", issuerId)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetRoot is a free data retrieval call binding the contract method 0x84f94221.
//
// Solidity: function getRoot(bytes32 issuerId) view returns(uint256)
func (_LeanIMTPlusRootStorage *LeanIMTPlusRootStorageSession) GetRoot(issuerId [32]byte) (*big.Int, error) {
	return _LeanIMTPlusRootStorage.Contract.GetRoot(&_LeanIMTPlusRootStorage.CallOpts, issuerId)
}

// GetRoot is a free data retrieval call binding the contract method 0x84f94221.
//
// Solidity: function getRoot(bytes32 issuerId) view returns(uint256)
func (_LeanIMTPlusRootStorage *LeanIMTPlusRootStorageCallerSession) GetRoot(issuerId [32]byte) (*big.Int, error) {
	return _LeanIMTPlusRootStorage.Contract.GetRoot(&_LeanIMTPlusRootStorage.CallOpts, issuerId)
}

// GetRootInfo is a free data retrieval call binding the contract method 0x41cc5bcb.
//
// Solidity: function getRootInfo(bytes32 issuerId) view returns(uint256 root, uint256 crlNumber, uint256 updatedAt, uint8 depth, uint64 leafCount)
func (_LeanIMTPlusRootStorage *LeanIMTPlusRootStorageCaller) GetRootInfo(opts *bind.CallOpts, issuerId [32]byte) (struct {
	Root      *big.Int
	CrlNumber *big.Int
	UpdatedAt *big.Int
	Depth     uint8
	LeafCount uint64
}, error) {
	var out []interface{}
	err := _LeanIMTPlusRootStorage.contract.Call(opts, &out, "getRootInfo", issuerId)

	outstruct := new(struct {
		Root      *big.Int
		CrlNumber *big.Int
		UpdatedAt *big.Int
		Depth     uint8
		LeafCount uint64
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Root = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.CrlNumber = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.UpdatedAt = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)
	outstruct.Depth = *abi.ConvertType(out[3], new(uint8)).(*uint8)
	outstruct.LeafCount = *abi.ConvertType(out[4], new(uint64)).(*uint64)

	return *outstruct, err

}

// GetRootInfo is a free data retrieval call binding the contract method 0x41cc5bcb.
//
// Solidity: function getRootInfo(bytes32 issuerId) view returns(uint256 root, uint256 crlNumber, uint256 updatedAt, uint8 depth, uint64 leafCount)
func (_LeanIMTPlusRootStorage *LeanIMTPlusRootStorageSession) GetRootInfo(issuerId [32]byte) (struct {
	Root      *big.Int
	CrlNumber *big.Int
	UpdatedAt *big.Int
	Depth     uint8
	LeafCount uint64
}, error) {
	return _LeanIMTPlusRootStorage.Contract.GetRootInfo(&_LeanIMTPlusRootStorage.CallOpts, issuerId)
}

// GetRootInfo is a free data retrieval call binding the contract method 0x41cc5bcb.
//
// Solidity: function getRootInfo(bytes32 issuerId) view returns(uint256 root, uint256 crlNumber, uint256 updatedAt, uint8 depth, uint64 leafCount)
func (_LeanIMTPlusRootStorage *LeanIMTPlusRootStorageCallerSession) GetRootInfo(issuerId [32]byte) (struct {
	Root      *big.Int
	CrlNumber *big.Int
	UpdatedAt *big.Int
	Depth     uint8
	LeafCount uint64
}, error) {
	return _LeanIMTPlusRootStorage.Contract.GetRootInfo(&_LeanIMTPlusRootStorage.CallOpts, issuerId)
}

// Relayer is a free data retrieval call binding the contract method 0x8406c079.
//
// Solidity: function relayer() view returns(address)
func (_LeanIMTPlusRootStorage *LeanIMTPlusRootStorageCaller) Relayer(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _LeanIMTPlusRootStorage.contract.Call(opts, &out, "relayer")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Relayer is a free data retrieval call binding the contract method 0x8406c079.
//
// Solidity: function relayer() view returns(address)
func (_LeanIMTPlusRootStorage *LeanIMTPlusRootStorageSession) Relayer() (common.Address, error) {
	return _LeanIMTPlusRootStorage.Contract.Relayer(&_LeanIMTPlusRootStorage.CallOpts)
}

// Relayer is a free data retrieval call binding the contract method 0x8406c079.
//
// Solidity: function relayer() view returns(address)
func (_LeanIMTPlusRootStorage *LeanIMTPlusRootStorageCallerSession) Relayer() (common.Address, error) {
	return _LeanIMTPlusRootStorage.Contract.Relayer(&_LeanIMTPlusRootStorage.CallOpts)
}

// Roots is a free data retrieval call binding the contract method 0xae6dead7.
//
// Solidity: function roots(bytes32 ) view returns(uint256 root, uint256 crlNumber, uint256 updatedAt, uint8 depth, uint64 leafCount)
func (_LeanIMTPlusRootStorage *LeanIMTPlusRootStorageCaller) Roots(opts *bind.CallOpts, arg0 [32]byte) (struct {
	Root      *big.Int
	CrlNumber *big.Int
	UpdatedAt *big.Int
	Depth     uint8
	LeafCount uint64
}, error) {
	var out []interface{}
	err := _LeanIMTPlusRootStorage.contract.Call(opts, &out, "roots", arg0)

	outstruct := new(struct {
		Root      *big.Int
		CrlNumber *big.Int
		UpdatedAt *big.Int
		Depth     uint8
		LeafCount uint64
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Root = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.CrlNumber = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.UpdatedAt = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)
	outstruct.Depth = *abi.ConvertType(out[3], new(uint8)).(*uint8)
	outstruct.LeafCount = *abi.ConvertType(out[4], new(uint64)).(*uint64)

	return *outstruct, err

}

// Roots is a free data retrieval call binding the contract method 0xae6dead7.
//
// Solidity: function roots(bytes32 ) view returns(uint256 root, uint256 crlNumber, uint256 updatedAt, uint8 depth, uint64 leafCount)
func (_LeanIMTPlusRootStorage *LeanIMTPlusRootStorageSession) Roots(arg0 [32]byte) (struct {
	Root      *big.Int
	CrlNumber *big.Int
	UpdatedAt *big.Int
	Depth     uint8
	LeafCount uint64
}, error) {
	return _LeanIMTPlusRootStorage.Contract.Roots(&_LeanIMTPlusRootStorage.CallOpts, arg0)
}

// Roots is a free data retrieval call binding the contract method 0xae6dead7.
//
// Solidity: function roots(bytes32 ) view returns(uint256 root, uint256 crlNumber, uint256 updatedAt, uint8 depth, uint64 leafCount)
func (_LeanIMTPlusRootStorage *LeanIMTPlusRootStorageCallerSession) Roots(arg0 [32]byte) (struct {
	Root      *big.Int
	CrlNumber *big.Int
	UpdatedAt *big.Int
	Depth     uint8
	LeafCount uint64
}, error) {
	return _LeanIMTPlusRootStorage.Contract.Roots(&_LeanIMTPlusRootStorage.CallOpts, arg0)
}

// SetRoot is a paid mutator transaction binding the contract method 0xb688289a.
//
// Solidity: function setRoot(bytes32 issuerId, uint256 newRoot, uint256 crlNumber, uint8 depth, uint64 leafCount) returns()
func (_LeanIMTPlusRootStorage *LeanIMTPlusRootStorageTransactor) SetRoot(opts *bind.TransactOpts, issuerId [32]byte, newRoot *big.Int, crlNumber *big.Int, depth uint8, leafCount uint64) (*types.Transaction, error) {
	return _LeanIMTPlusRootStorage.contract.Transact(opts, "setRoot", issuerId, newRoot, crlNumber, depth, leafCount)
}

// SetRoot is a paid mutator transaction binding the contract method 0xb688289a.
//
// Solidity: function setRoot(bytes32 issuerId, uint256 newRoot, uint256 crlNumber, uint8 depth, uint64 leafCount) returns()
func (_LeanIMTPlusRootStorage *LeanIMTPlusRootStorageSession) SetRoot(issuerId [32]byte, newRoot *big.Int, crlNumber *big.Int, depth uint8, leafCount uint64) (*types.Transaction, error) {
	return _LeanIMTPlusRootStorage.Contract.SetRoot(&_LeanIMTPlusRootStorage.TransactOpts, issuerId, newRoot, crlNumber, depth, leafCount)
}

// SetRoot is a paid mutator transaction binding the contract method 0xb688289a.
//
// Solidity: function setRoot(bytes32 issuerId, uint256 newRoot, uint256 crlNumber, uint8 depth, uint64 leafCount) returns()
func (_LeanIMTPlusRootStorage *LeanIMTPlusRootStorageTransactorSession) SetRoot(issuerId [32]byte, newRoot *big.Int, crlNumber *big.Int, depth uint8, leafCount uint64) (*types.Transaction, error) {
	return _LeanIMTPlusRootStorage.Contract.SetRoot(&_LeanIMTPlusRootStorage.TransactOpts, issuerId, newRoot, crlNumber, depth, leafCount)
}

// LeanIMTPlusRootStorageRootUpdatedIterator is returned from FilterRootUpdated and is used to iterate over the raw logs and unpacked data for RootUpdated events raised by the LeanIMTPlusRootStorage contract.
type LeanIMTPlusRootStorageRootUpdatedIterator struct {
	Event *LeanIMTPlusRootStorageRootUpdated // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *LeanIMTPlusRootStorageRootUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(LeanIMTPlusRootStorageRootUpdated)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(LeanIMTPlusRootStorageRootUpdated)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *LeanIMTPlusRootStorageRootUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *LeanIMTPlusRootStorageRootUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// LeanIMTPlusRootStorageRootUpdated represents a RootUpdated event raised by the LeanIMTPlusRootStorage contract.
type LeanIMTPlusRootStorageRootUpdated struct {
	IssuerId  [32]byte
	Root      *big.Int
	CrlNumber *big.Int
	Depth     uint8
	LeafCount uint64
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterRootUpdated is a free log retrieval operation binding the contract event 0xecccef0c55a00dc23786b6780a5499d379220ff8f6f9a14c408f0b6546f00735.
//
// Solidity: event RootUpdated(bytes32 indexed issuerId, uint256 root, uint256 crlNumber, uint8 depth, uint64 leafCount)
func (_LeanIMTPlusRootStorage *LeanIMTPlusRootStorageFilterer) FilterRootUpdated(opts *bind.FilterOpts, issuerId [][32]byte) (*LeanIMTPlusRootStorageRootUpdatedIterator, error) {

	var issuerIdRule []interface{}
	for _, issuerIdItem := range issuerId {
		issuerIdRule = append(issuerIdRule, issuerIdItem)
	}

	logs, sub, err := _LeanIMTPlusRootStorage.contract.FilterLogs(opts, "RootUpdated", issuerIdRule)
	if err != nil {
		return nil, err
	}
	return &LeanIMTPlusRootStorageRootUpdatedIterator{contract: _LeanIMTPlusRootStorage.contract, event: "RootUpdated", logs: logs, sub: sub}, nil
}

// WatchRootUpdated is a free log subscription operation binding the contract event 0xecccef0c55a00dc23786b6780a5499d379220ff8f6f9a14c408f0b6546f00735.
//
// Solidity: event RootUpdated(bytes32 indexed issuerId, uint256 root, uint256 crlNumber, uint8 depth, uint64 leafCount)
func (_LeanIMTPlusRootStorage *LeanIMTPlusRootStorageFilterer) WatchRootUpdated(opts *bind.WatchOpts, sink chan<- *LeanIMTPlusRootStorageRootUpdated, issuerId [][32]byte) (event.Subscription, error) {

	var issuerIdRule []interface{}
	for _, issuerIdItem := range issuerId {
		issuerIdRule = append(issuerIdRule, issuerIdItem)
	}

	logs, sub, err := _LeanIMTPlusRootStorage.contract.WatchLogs(opts, "RootUpdated", issuerIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(LeanIMTPlusRootStorageRootUpdated)
				if err := _LeanIMTPlusRootStorage.contract.UnpackLog(event, "RootUpdated", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseRootUpdated is a log parse operation binding the contract event 0xecccef0c55a00dc23786b6780a5499d379220ff8f6f9a14c408f0b6546f00735.
//
// Solidity: event RootUpdated(bytes32 indexed issuerId, uint256 root, uint256 crlNumber, uint8 depth, uint64 leafCount)
func (_LeanIMTPlusRootStorage *LeanIMTPlusRootStorageFilterer) ParseRootUpdated(log types.Log) (*LeanIMTPlusRootStorageRootUpdated, error) {
	event := new(LeanIMTPlusRootStorageRootUpdated)
	if err := _LeanIMTPlusRootStorage.contract.UnpackLog(event, "RootUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
