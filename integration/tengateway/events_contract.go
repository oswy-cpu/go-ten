package tengateway

import (
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

func init() { //nolint: gochecknoinits
	contractABI, err := abi.JSON(strings.NewReader(eventsContractABIString))
	if err != nil {
		panic(err)
	}
	eventsContractABI = contractABI
}

var (
	eventsContractABI       abi.ABI
	eventsContractBytecode  = "0x60806040523480156200001157600080fd5b506040518060400160405280600381526020017f666f6f00000000000000000000000000000000000000000000000000000000008152506000908162000058919062000320565b506040518060400160405280600381526020017f666f6f0000000000000000000000000000000000000000000000000000000000815250600190816200009f919062000320565b5062000407565b600081519050919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b7f4e487b7100000000000000000000000000000000000000000000000000000000600052602260045260246000fd5b600060028204905060018216806200012857607f821691505b6020821081036200013e576200013d620000e0565b5b50919050565b60008190508160005260206000209050919050565b60006020601f8301049050919050565b600082821b905092915050565b600060088302620001a87fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8262000169565b620001b4868362000169565b95508019841693508086168417925050509392505050565b6000819050919050565b6000819050919050565b600062000201620001fb620001f584620001cc565b620001d6565b620001cc565b9050919050565b6000819050919050565b6200021d83620001e0565b620002356200022c8262000208565b84845462000176565b825550505050565b600090565b6200024c6200023d565b6200025981848462000212565b505050565b5b8181101562000281576200027560008262000242565b6001810190506200025f565b5050565b601f821115620002d0576200029a8162000144565b620002a58462000159565b81016020851015620002b5578190505b620002cd620002c48562000159565b8301826200025e565b50505b505050565b600082821c905092915050565b6000620002f560001984600802620002d5565b1980831691505092915050565b6000620003108383620002e2565b9150826002028217905092915050565b6200032b82620000a6565b67ffffffffffffffff811115620003475762000346620000b1565b5b6200035382546200010f565b6200036082828562000285565b600060209050601f83116001811462000398576000841562000383578287015190505b6200038f858262000302565b865550620003ff565b601f198416620003a88662000144565b60005b82811015620003d257848901518255600182019150602085019450602081019050620003ab565b86831015620003f25784890151620003ee601f891682620002e2565b8355505b6001600288020188555050505b505050505050565b6107ee80620004176000396000f3fe608060405234801561001057600080fd5b506004361061004c5760003560e01c8063368b877214610051578063c2d366581461006d578063c5ced0361461008b578063e21f37ce146100a7575b600080fd5b61006b600480360381019061006691906103e6565b6100c5565b005b610075610126565b60405161008291906104ae565b60405180910390f35b6100a560048036038101906100a091906103e6565b6101b4565b005b6100af6101fe565b6040516100bc91906104ae565b60405180910390f35b80600090816100d491906106e6565b503373ffffffffffffffffffffffffffffffffffffffff167fe31c2ad953ded70272b94617f9181f8cc33755f1b40f4c706660f6ee0dfb634a8260405161011b91906104ae565b60405180910390a250565b60018054610133906104ff565b80601f016020809104026020016040519081016040528092919081815260200182805461015f906104ff565b80156101ac5780601f10610181576101008083540402835291602001916101ac565b820191906000526020600020905b81548152906001019060200180831161018f57829003601f168201915b505050505081565b80600190816101c391906106e6565b507f4fcdf2659dcf2254d2bce07af2baaf0c6ddf6da052dd241b2445a2f6398ae575816040516101f391906104ae565b60405180910390a150565b6000805461020b906104ff565b80601f0160208091040260200160405190810160405280929190818152602001828054610237906104ff565b80156102845780601f1061025957610100808354040283529160200191610284565b820191906000526020600020905b81548152906001019060200180831161026757829003601f168201915b505050505081565b6000604051905090565b600080fd5b600080fd5b600080fd5b600080fd5b6000601f19601f8301169050919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b6102f3826102aa565b810181811067ffffffffffffffff82111715610312576103116102bb565b5b80604052505050565b600061032561028c565b905061033182826102ea565b919050565b600067ffffffffffffffff821115610351576103506102bb565b5b61035a826102aa565b9050602081019050919050565b82818337600083830152505050565b600061038961038484610336565b61031b565b9050828152602081018484840111156103a5576103a46102a5565b5b6103b0848285610367565b509392505050565b600082601f8301126103cd576103cc6102a0565b5b81356103dd848260208601610376565b91505092915050565b6000602082840312156103fc576103fb610296565b5b600082013567ffffffffffffffff81111561041a5761041961029b565b5b610426848285016103b8565b91505092915050565b600081519050919050565b600082825260208201905092915050565b60005b8381101561046957808201518184015260208101905061044e565b60008484015250505050565b60006104808261042f565b61048a818561043a565b935061049a81856020860161044b565b6104a3816102aa565b840191505092915050565b600060208201905081810360008301526104c88184610475565b905092915050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052602260045260246000fd5b6000600282049050600182168061051757607f821691505b60208210810361052a576105296104d0565b5b50919050565b60008190508160005260206000209050919050565b60006020601f8301049050919050565b600082821b905092915050565b6000600883026105927fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff82610555565b61059c8683610555565b95508019841693508086168417925050509392505050565b6000819050919050565b6000819050919050565b60006105e36105de6105d9846105b4565b6105be565b6105b4565b9050919050565b6000819050919050565b6105fd836105c8565b610611610609826105ea565b848454610562565b825550505050565b600090565b610626610619565b6106318184846105f4565b505050565b5b818110156106555761064a60008261061e565b600181019050610637565b5050565b601f82111561069a5761066b81610530565b61067484610545565b81016020851015610683578190505b61069761068f85610545565b830182610636565b50505b505050565b600082821c905092915050565b60006106bd6000198460080261069f565b1980831691505092915050565b60006106d683836106ac565b9150826002028217905092915050565b6106ef8261042f565b67ffffffffffffffff811115610708576107076102bb565b5b61071282546104ff565b61071d828285610659565b600060209050601f831160018114610750576000841561073e578287015190505b61074885826106ca565b8655506107b0565b601f19841661075e86610530565b60005b8281101561078657848901518255600182019150602085019450602081019050610761565b868310156107a3578489015161079f601f8916826106ac565b8355505b6001600288020188555050505b50505050505056fea264697066735822122076146d8c796917af248ecb981f38094293788d92ad21704dc623fd8412cb9dc964736f6c63430008120033"
	eventsContractABIString = `
	[
	{
		"inputs": [],
		"stateMutability": "nonpayable",
		"type": "constructor"
	},
	{
		"anonymous": false,
		"inputs": [
			{
				"indexed": false,
				"internalType": "string",
				"name": "newMessage",
				"type": "string"
			}
		],
		"name": "Message2Updated",
		"type": "event"
	},
	{
		"anonymous": false,
		"inputs": [
			{
				"indexed": false,
				"internalType": "string",
				"name": "newMessage",
				"type": "string"
			},
			{
				"indexed": true,
				"internalType": "address",
				"name": "sender",
				"type": "address"
			}
		],
		"name": "MessageUpdatedWithAddress",
		"type": "event"
	},
	{
		"inputs": [],
		"name": "message",
		"outputs": [
			{
				"internalType": "string",
				"name": "",
				"type": "string"
			}
		],
		"stateMutability": "view",
		"type": "function"
	},
	{
		"inputs": [],
		"name": "message2",
		"outputs": [
			{
				"internalType": "string",
				"name": "",
				"type": "string"
			}
		],
		"stateMutability": "view",
		"type": "function"
	},
	{
		"inputs": [
			{
				"internalType": "string",
				"name": "newMessage",
				"type": "string"
			}
		],
		"name": "setMessage",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"inputs": [
			{
				"internalType": "string",
				"name": "newMessage",
				"type": "string"
			}
		],
		"name": "setMessage2",
		"outputs": [],
		"stateMutability": "nonpayable",
		"type": "function"
	}
]
	`
)

//// SPDX-License-Identifier: MIT
//pragma solidity ^0.8.0;
//
//contract SimpleMessageContract {
//
//// State variable to store the message
//string public message;
//string public message2;
//
//// Event declaration
//event MessageUpdatedWithAddress(string newMessage, address indexed sender);
//event Message2Updated(string newMessage);
//
//// Constructor to initialize the message
//constructor() {
//message = "foo";
//message2 = "foo";
//}
//
//// Function to set a new message
//function setMessage(string memory newMessage) public {
//message = newMessage;
//emit MessageUpdatedWithAddress(newMessage, msg.sender);  // Emit the event (only sender can see it)
//}
//
//function setMessage2(string memory newMessage) public {
//message2 = newMessage;
//emit Message2Updated(newMessage);  // Emit the event (everyone can see it)
//}
//}