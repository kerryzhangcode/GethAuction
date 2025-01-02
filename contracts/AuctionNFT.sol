pragma solidity ^0.8.28;

import {ERC721URIStorage, ERC721} from "@openzeppelin/contracts/token/ERC721/extensions/ERC721URIStorage.sol";
import { Ownable } from "@openzeppelin/contracts/access/Ownable.sol";

contract AuctionNFT is ERC721URIStorage, Ownable {
    uint private _tokenId;

    constructor() ERC721("AuctionNFT", "AUC") Ownable(msg.sender) {
        _tokenId = 0;
    }

    function mint(address to, string memory tokenURI) public onlyOwner returns (uint) {
        _tokenId++;
        _mint(to, _tokenId);
        _setTokenURI(_tokenId, tokenURI);

        return _tokenId;
    }
}