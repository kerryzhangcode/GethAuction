// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.28;

// Uncomment this line to use console.log
// import "hardhat/console.sol";

import "@openzeppelin/contracts/utils/math/Math.sol";
import { Ownable } from "@openzeppelin/contracts/access/Ownable.sol";
import { IERC721Receiver } from "@openzeppelin/contracts/token/ERC721/IERC721Receiver.sol";

import "./AuctionNFT.sol";

contract Auction is IERC721Receiver, Ownable {
    enum State {Preparation, Running, Ended, Withdrawed}
    uint private _currentRecordId = 0;
    struct AuctionRecord {
        uint recordId;
        uint lowestPrice;
        uint bidIncrement;
        uint startTimestamp;
        uint endTimestamp;
        State state;
        uint highestBid;
        address highestBidder;
    }
    mapping(uint => AuctionRecord) public auctions;
    struct Bid {
        address bidder;
        uint amount;
        uint timestamp;
    }
    mapping(uint => Bid[]) public bidsHistory;
    mapping(address => mapping (uint => uint)) public userItemBids;
    mapping(uint => address[]) public itemBidders;
    mapping(address => uint256[]) public receivedTokens;

    AuctionNFT public auctionNFT;  // AuctionNFT Contract Instance

    constructor(address auctionNFTAddress) Ownable(msg.sender) {
        auctionNFT = AuctionNFT(auctionNFTAddress);
    }

    function onERC721Received(address operator, address from, uint256 tokenId, bytes calldata data) external override returns (bytes4) {
        receivedTokens[from].push(tokenId);
        return this.onERC721Received.selector;
    }

    function startAuction(uint tokenId, uint _lowestPrice, uint _bidIncrement, uint _endTimestamp) public returns (uint) {
        // Check if the caller transfers the token to the contract
        require(auctionNFT.ownerOf(tokenId) == address(this), "Token is not transferred to the contract");
        // Check if the caller is the owner of the token
        for (uint i = 0; i < receivedTokens[msg.sender].length; i++) {
            if (receivedTokens[msg.sender][i] == tokenId) {
                break;
            }
            if (i == receivedTokens[msg.sender].length - 1) {
                revert("You are not the owner of the token");
            }
        }  

        // Check if the auction is not existing
        require(auctions[tokenId].state == State.Preparation, "Auction is existing");

        require(_lowestPrice > 0, "Lowest price must be greater than 0");
        require(_bidIncrement > 0, "Bid increment must be greater than 0");
        require(_endTimestamp > block.timestamp, "End timestamp must be greater than current timestamp");

        _currentRecordId++;
        auctions[tokenId] = AuctionRecord({
            recordId: _currentRecordId,
            lowestPrice: _lowestPrice,
            bidIncrement: _bidIncrement,
            startTimestamp: block.timestamp,
            endTimestamp: _endTimestamp,
            state: State.Running,
            highestBid: 0,
            highestBidder: msg.sender
        });
        return _currentRecordId;
    }

    function _checkAuctionExist (uint tokenId) private view {
        require(auctions[tokenId].state != State.Preparation, "Auction is not existing");
    }

    modifier checkAuctionExist(uint tokenId) {
        _checkAuctionExist(tokenId);
        _;
    }

    modifier checkAuctionRunning(uint tokenId) {
        _checkAuctionExist(tokenId);
        // Check if the auction is ended
        require(auctions[tokenId].state != State.Ended, "Auction is ended");
        // Check if the auction is running
        require(auctions[tokenId].state == State.Running, "Auction is not running");
        _;
    }

    function incrementBid(uint tokenId) public payable checkAuctionRunning(tokenId) {
        // Check if the auction can be ended 
        bool flag = _endAuction(tokenId);
        require(!flag, "Auction is ended");

        // Check bid legitimacy
        require(msg.value >= auctions[tokenId].bidIncrement, "Bid is lower than the bid increment");
        uint previousBid = userItemBids[msg.sender][tokenId];
        // uint currentBid = previousBid + msg.value;
        (bool success, uint currentBid) = Math.tryAdd(previousBid, msg.value);
        require(success, "Bid is too high");
        require(currentBid >= auctions[tokenId].lowestPrice, "Bid is lower than the lowest price");
        require(currentBid > auctions[tokenId].highestBid, "Bid is lower than the highest bid");

        // Update the bidders
        if(previousBid == 0) {
            itemBidders[tokenId].push(msg.sender);
        }
        // Update the highest bid of the item
        auctions[tokenId].highestBid = currentBid;
        auctions[tokenId].highestBidder = msg.sender;
        // Update the user's bid for the item
        userItemBids[msg.sender][tokenId] = currentBid;
        // Update the bid history of the item
        Bid memory newBid = Bid({
            bidder: msg.sender,
            amount: currentBid,
            timestamp: block.timestamp
        });
        bidsHistory[tokenId].push(newBid);
    }

    function endAuction(uint tokenId) public checkAuctionRunning(tokenId) {
        bool flag = _endAuction(tokenId);
        require(flag, "Auction can't be ended");
    }

    function _endAuction(uint tokenId) private returns (bool) {
        // Check if the auction can be ended
        if (block.timestamp < auctions[tokenId].endTimestamp) {
            return false;
        }

        auctions[tokenId].state = State.Ended;
        sendMoneyToOwner(auctions[tokenId].highestBid);

        // Update the item bidders
        address winner = auctions[tokenId].highestBidder;
        userItemBids[winner][tokenId] = 0;
        _removeAddressFromBidders(tokenId, winner);

        // Send the item to the highest bidder
        auctionNFT.safeTransferFrom(address(this), winner, tokenId);
        for (uint i = 0; i < receivedTokens[winner].length; i++) {
            if (receivedTokens[winner][i] == tokenId) {
                receivedTokens[winner][i] = receivedTokens[winner][receivedTokens[winner].length - 1];
                receivedTokens[winner].pop();
                break;
            }
        }

        _resetAuction(tokenId);
        return true;
    }

    function _removeAddressFromBidders(uint tokenId, address adr) private returns (bool) {
        address[] storage _itemBidders = itemBidders[tokenId];
        for (uint256 i = 0; i < _itemBidders.length; i++) {
            if (_itemBidders[i] == adr) {
                _itemBidders[i] = _itemBidders[_itemBidders.length - 1];
                _itemBidders.pop();
                return true;
            }
        }
        return false;
    }

    function withdraw(uint tokenId) public checkAuctionExist(tokenId) {
        // Check if the auction is ended
        require(auctions[tokenId].state == State.Ended, "Auction is not ended");

        // Check if the user is the highest bidder
        require(msg.sender != auctions[tokenId].highestBidder, "You are the highest bidder");
        // Check if the user has a bid
        require(userItemBids[msg.sender][tokenId] > 0, "You don't have a bid");

        uint amount = userItemBids[msg.sender][tokenId];
        userItemBids[msg.sender][tokenId] = 0;

        // Update the item bidder
        _removeAddressFromBidders(tokenId, msg.sender);
        _resetAuction(tokenId);
        
        payable(msg.sender).transfer(amount);
    }

    function _resetAuction(uint tokenId) private checkAuctionExist(tokenId) {
        // Check if the auction is ended
        if(auctions[tokenId].state != State.Ended) {
            return;
        }
        if(itemBidders[tokenId].length > 0) {
            return;
        }

        auctions[tokenId].state = State.Preparation;
        auctions[tokenId].highestBid = 0;
        auctions[tokenId].highestBidder = address(0);
    }

    function initializeAuction(uint tokenId) public onlyOwner checkAuctionExist(tokenId) {
        require(auctions[tokenId].state == State.Ended, "Auction is not ended");

        auctions[tokenId].state = State.Preparation;
        auctions[tokenId].highestBid = 0;
        auctions[tokenId].highestBidder = address(0);

        uint remainAmount = 0;
        address[] memory bidders = itemBidders[tokenId];
        for (uint i = 0; i < bidders.length; i++) {
            address bidder = bidders[i];
            uint amount = userItemBids[bidder][tokenId];
            userItemBids[bidder][tokenId] = 0;
            remainAmount += amount;
        }
        itemBidders[tokenId] = new address[](0);

        if(remainAmount > 0) {
            payable(owner()).transfer(remainAmount);
        }    
    }

    function sendMoneyToOwner(uint amount) private {
        payable(owner()).transfer(amount);
    }

    function getBidsHistory(uint itemId) public view returns (Bid[] memory) {
        return bidsHistory[itemId];
    }

    function getHighestBid(uint itemId) public view returns (uint) {
        return auctions[itemId].highestBid;
    }

    function getHighestBidder(uint itemId) public view returns (address) {
        return auctions[itemId].highestBidder;
    }

    function getAuctionRecord(uint itemId) public view returns (AuctionRecord memory) {
        return auctions[itemId];
    }

    function getBid(uint itemId) public view returns (uint) {
        return userItemBids[msg.sender][itemId];
    }

    function getBalance() public view returns (uint) {
        return address(this).balance;
    }

    function getReceivedTokens(address user) public view returns (uint256[] memory) {
        return receivedTokens[user];
    }
}